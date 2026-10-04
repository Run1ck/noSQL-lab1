// Package redisstore — хранение временных данных приложения в Redis.
package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// New подключается к Redis и проверяет соединение PING'ом, чтобы
// приложение падало при старте, а не на первом запросе.
func New(ctx context.Context, addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis %s: %w", addr, err)
	}
	return rdb, nil
}
