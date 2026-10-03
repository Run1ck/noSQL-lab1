package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT string

	RedisAddr string

	CartTTL      time.Duration
	CacheTTL     time.Duration
	CacheEnabled bool

	RateLimit  int
	RateWindow time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load("deploy/.env", ".env")

	cfg := Config{
		PORT:         env("PORT", ":8080"),
		CartTTL:      envDuration("CART_TTL", 30*time.Minute),
		CacheTTL:     envDuration("CACHE_TTL", 60*time.Second),
		CacheEnabled: envBool("CACHE_ENABLED", true),
		RateLimit:    envInt("RATE_LIMIT", 60),
		RateWindow:   envDuration("RATE_WINDOW", time.Minute),
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
