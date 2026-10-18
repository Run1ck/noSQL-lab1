package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type cache struct {
	redis *redis.Client
	ttl   time.Duration
}

func (c cache) load(ctx context.Context, key string, dst any) bool {
	data, err := c.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false
	}
	if err != nil {
		slog.Warn("cache read failed, falling back to storage", "key", key, "err", err)

		return false
	}

	err = json.Unmarshal(data, dst)
	if err != nil {
		slog.Warn("cache entry is corrupted", "key", key, "err", err)

		return false
	}

	return true
}

func (c cache) store(ctx context.Context, key string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Warn("cache encode failed", "key", key, "err", err)

		return
	}

	err = c.redis.Set(ctx, key, data, c.ttl).Err()
	if err != nil {
		slog.Warn("cache write failed", "key", key, "err", err)
	}
}

func (c cache) invalidate(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}

	err := c.redis.Del(ctx, keys...).Err()
	if err != nil {
		slog.Warn("cache invalidation failed, entries stay until TTL", "keys", keys, "err", err)
	}
}
