package repository

import (
	"time"

	"github.com/redis/go-redis/v9"

	"booking/internal/usecase"
)

// Кэш поверх Postgres: список всех услуг и брони дня — JSON-строки с TTL.
// Сбрасываются при изменении: SaveService — услуги, Approve — дни заявки.
const (
	servicesKey    = "cache:services"
	schedulePrefix = "cache:schedule:"
)

type Repository struct {
	redis    *redis.Client
	postgres usecase.Postgres
	ttl      time.Duration
}

func New(client *redis.Client, p usecase.Postgres, ttl time.Duration) *Repository {
	return &Repository{
		redis:    client,
		postgres: p,
		ttl:      ttl,
	}
}
