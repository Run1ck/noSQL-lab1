// Package postgres — постоянное хранение данных приложения в PostgreSQL.
// Запросы генерирует sqlc в подпакет sqlc (postgres/sqlc.yaml).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New создаёт пул соединений и проверяет базу Ping'ом: пул подключается
// лениво, и без проверки приложение падало бы на первом запросе, а не при
// старте. В ошибку не попадает dsn — в нём пароль.
func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
