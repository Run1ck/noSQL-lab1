package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"booking/internal/domain/service"
)

var errCacheMiss = errors.New("cache miss")

func scheduleKey(date service.Date) string {
	return schedulePrefix + date.String()
}

func (r *Repository) getCache(ctx context.Context, key string, v any) error {
	data, err := r.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errCacheMiss
		}

		return fmt.Errorf("r.redis.Get: %w", err)
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	return nil
}

func (r *Repository) setCache(ctx context.Context, key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	err = r.redis.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("r.redis.Set: %w", err)
	}

	return nil
}

func (r *Repository) deleteCache(ctx context.Context, keys ...string) error {
	err := r.redis.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("r.redis.Del: %w", err)
	}

	return nil
}
