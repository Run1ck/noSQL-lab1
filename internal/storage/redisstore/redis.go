// Package redisstore — хранение временных данных приложения в Redis.
package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// timeout короче дефолтов go-redis (подключение и ожидание ответа по 5 с):
// Redis локальный и отвечает за доли миллисекунды, а при его отказе лимит API
// пропускает запросы (fail-open) — с дефолтами каждый такой запрос ждал бы
// ~1,7 с, а если Redis пропал из сети, то десятки секунд.
const timeout = 500 * time.Millisecond

// New подключается к Redis и проверяет соединение PING'ом, чтобы
// приложение падало при старте, а не на первом запросе.
func New(ctx context.Context, addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:        addr,
		DialTimeout: timeout,
		// Это число попыток подключения, а не повторов после первой, хотя
		// go-redis называет поле «retry attempts». 0 — дефолт, 5 попыток.
		DialerRetries: 1,
		ReadTimeout:   timeout,
		WriteTimeout:  timeout,
		// Без повторов команд. Если ответ не пришёл за ReadTimeout, команда,
		// возможно, уже выполнена, и повтор выполнил бы её второй раз: скрипт
		// лимита посчитал бы одну заявку дважды и выдал ложный бан. Мёртвые
		// соединения после рестарта Redis пул отсеивает сам при выдаче.
		MaxRetries: -1,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis %s: %w", addr, err)
	}
	return rdb, nil
}
