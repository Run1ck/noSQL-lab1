package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"booking/internal/domain/booking"
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
)

//go:generate mockery

type Redis interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error)
	SaveCart(ctx context.Context, c *cart.Cart) error
	RemoveCartItem(ctx context.Context, userID uuid.UUID, item cart.Item) error
	DeleteCart(ctx context.Context, userID uuid.UUID) error
	GetCartTTL(ctx context.Context, userID uuid.UUID) (time.Duration, error)
}

type Postgres interface {
	GetServices(ctx context.Context) ([]*service.Service, error)
	GetService(ctx context.Context, id string) (*service.Service, error)
	GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error)
}

type UseCase struct {
	postgres Postgres
	redis    Redis
}

func New(postgres Postgres, redis Redis) *UseCase {
	return &UseCase{
		postgres: postgres,
		redis:    redis,
	}
}
