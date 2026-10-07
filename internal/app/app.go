package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"booking/config"
	adapterpostgres "booking/internal/adapter/postgres"
	adapterredis "booking/internal/adapter/redis"
	"booking/internal/auth"
	controllerhttp "booking/internal/controller/http"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/pkg/httpx"
	"booking/pkg/postgres"
	"booking/pkg/ratelimit"
	"booking/pkg/redis"
)

const (
	startTimeout    = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

func Run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()

	pool, err := postgres.New(startCtx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("postgres.New: %w", err)
	}
	defer pool.Close()

	client, err := redis.New(startCtx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("redis.New: %w", err)
	}
	defer client.Close()

	var pg usecase.Postgres = adapterpostgres.New(pool)
	tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	apiLimit := ratelimit.NewRedis(client, "rl:api", cfg.RateLimit, cfg.RateWindow)
	banLimit := ratelimit.NewBan(client, cfg.BanLimit, cfg.BanWindow, cfg.BanTTL)

	uc := usecase.New(pg, adapterredis.New(client, cfg.CartTTL), tokens, banLimit)

	admin, err := uc.EnsureAdmin(startCtx, dto.EnsureAdminInput{Login: cfg.AdminLogin, Password: cfg.AdminPassword})
	if err != nil {
		return fmt.Errorf("uc.EnsureAdmin: %w", err)
	}
	if admin.Created {
		slog.Info("admin created", "login", cfg.AdminLogin)
	}

	mux := http.NewServeMux()
	controllerhttp.Router(mux, uc, httpx.NewMiddlewares(tokens))

	srv := &http.Server{
		Addr:              cfg.PORT,
		Handler:           httpx.RateLimit(apiLimit, httpx.ClientIP)(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server started", "addr", cfg.PORT)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("srv.ListenAndServe: %w", err)
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("srv.Shutdown: %w", err)
	}

	return nil
}
