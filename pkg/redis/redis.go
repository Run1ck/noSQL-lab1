package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const timeout = 500 * time.Millisecond

func New(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:          addr,
		DialTimeout:   timeout,
		DialerRetries: 1,
		ReadTimeout:   timeout,
		WriteTimeout:  timeout,
		MaxRetries:    -1,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("client.Ping: %w", err)
	}

	return client, nil
}
