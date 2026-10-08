package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"booking/config"
	adapterpostgres "booking/internal/adapter/postgres"
	adapterredis "booking/internal/adapter/redis"
	"booking/internal/adapter/repository"
	"booking/internal/auth"
	controllerhttp "booking/internal/controller/http"
	"booking/internal/usecase"
	"booking/pkg/httpx"
	"booking/pkg/postgres"
	"booking/pkg/ratelimit"
	"booking/pkg/redis"
)

func Run(ctx context.Context, c config.Config) error {
	// Redis
	redisClient, err := redis.New(ctx, c.RedisAddr)
	if err != nil {
		return fmt.Errorf("redis.New: %w", err)
	}

	// Postgres
	pgPool, err := postgres.New(ctx, c.PostgresDSN)
	if err != nil {
		return fmt.Errorf("postgres.New: %w", err)
	}

	var pg usecase.Postgres = adapterpostgres.New(pgPool)
	if c.CacheEnabled {
		pg = repository.New(redisClient, pg, c.CacheTTL)
	}

	// UseCase
	tokens := auth.NewJWT(c.JWTSecret, c.JWTTTL)
	ban := ratelimit.NewBan(redisClient, c.BanLimit, c.BanWindow, c.BanTTL)
	uc := usecase.New(pg, adapterredis.New(redisClient, c.CartTTL), tokens, ban)

	// HTTP
	mux := http.NewServeMux()
	controllerhttp.Router(mux, uc, httpx.NewMiddlewares(tokens))

	// Лимит только на /api/*, статику UI не считаем.
	apiLimit := ratelimit.NewRedis(redisClient, "rl:api", c.RateLimit, c.RateWindow)

	httpServer := &http.Server{
		Addr:              c.PORT,
		Handler:           httpx.RateLimit(apiLimit, httpx.ClientIP)(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("httpServer.ListenAndServe", "err", err)
		}
	}()

	slog.Info("App started!", "addr", c.PORT, "cache", c.CacheEnabled)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig // wait signal

	slog.Info("App got signal to stop")

	// Controllers
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = httpServer.Shutdown(shutdownCtx)
	if err != nil {
		slog.Error("httpServer.Shutdown", "err", err)
	}

	// Adapters
	_ = redisClient.Close()
	pgPool.Close()

	slog.Info("App stopped!")

	return nil
}
