package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// timeout короче дефолтов go-redis (подключение и ожидание ответа по 5 с):
// Redis локальный, а при его отказе лимит API пропускает запросы (fail-open) —
// с дефолтами каждый такой запрос ждал бы ~1,7 с.
const timeout = 500 * time.Millisecond

// New подключается к Redis и проверяет соединение PING'ом, чтобы приложение
// падало при старте, а не на первом запросе.
func New(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		DialTimeout: timeout,
		// Число попыток подключения, а не повторов; 0 — дефолт, 5 попыток.
		DialerRetries: 1,
		ReadTimeout:   timeout,
		WriteTimeout:  timeout,
		// Без повторов: после таймаута чтения команда, возможно, уже
		// выполнена, и повтор посчитал бы Lua-скрипт лимита дважды.
		MaxRetries: -1,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("client.Ping: %w", err)
	}

	return client, nil
}
