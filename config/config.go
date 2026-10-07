package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// defaultPostgresDSN совпадает с сервисом postgres в deploy/compose.yaml.
const defaultPostgresDSN = "postgres://booking:booking@127.0.0.1:5432/booking?sslmode=disable"

type Config struct {
	PORT string

	RedisAddr string

	CartTTL      time.Duration
	CacheTTL     time.Duration
	CacheEnabled bool

	RateLimit  int
	RateWindow time.Duration

	// Бан за частые заявки: больше BanLimit заявок за BanWindow — бан на BanTTL.
	BanLimit  int
	BanWindow time.Duration
	BanTTL    time.Duration

	PostgresDSN string

	JWTSecret string
	JWTTTL    time.Duration

	// Администратор, которого приложение создаёт при старте, если его ещё
	// нет. Пустой пароль — не создавать.
	AdminLogin    string
	AdminPassword string
}

func Load() (Config, error) {
	_ = godotenv.Load("deploy/.env", ".env")

	cfg := Config{
		PORT:          env("PORT", ":8080"),
		CartTTL:       envDuration("CART_TTL", 30*time.Minute),
		CacheTTL:      envDuration("CACHE_TTL", 60*time.Second),
		CacheEnabled:  envBool("CACHE_ENABLED", true),
		RateLimit:     envInt("RATE_LIMIT", 60),
		RateWindow:    envDuration("RATE_WINDOW", time.Minute),
		BanLimit:      envInt("BAN_LIMIT", 5),
		BanWindow:     envDuration("BAN_WINDOW", 10*time.Minute),
		BanTTL:        envDuration("BAN_TTL", 15*time.Minute),
		PostgresDSN:   env("POSTGRES_DSN", defaultPostgresDSN),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTTTL:        envDuration("JWT_TTL", 24*time.Hour),
		AdminLogin:    env("ADMIN_LOGIN", "admin"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}
	cfg.RedisAddr = net.JoinHostPort(
		env("REDIS_HOST", "127.0.0.1"),
		env("REDIS_PORT", "6379"),
	)

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.CartTTL <= 0 {
		return fmt.Errorf("CART_TTL must be positive, got %s", c.CartTTL)
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("CACHE_TTL must be positive, got %s", c.CacheTTL)
	}
	if c.RateLimit <= 0 || c.RateWindow <= 0 {
		return fmt.Errorf("RATE_LIMIT and RATE_WINDOW must be positive, got %d per %s", c.RateLimit, c.RateWindow)
	}
	if c.BanLimit <= 0 || c.BanWindow <= 0 || c.BanTTL <= 0 {
		return fmt.Errorf("BAN_LIMIT, BAN_WINDOW and BAN_TTL must be positive, got %d per %s for %s", c.BanLimit, c.BanWindow, c.BanTTL)
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.JWTTTL <= 0 {
		return fmt.Errorf("JWT_TTL must be positive, got %s", c.JWTTTL)
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func envBool(key string, def bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func envDuration(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return def
	}
	return d
}
