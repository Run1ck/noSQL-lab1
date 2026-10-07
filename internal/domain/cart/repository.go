package cart

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Get(ctx context.Context, userID uuid.UUID) (*Cart, error)
	Save(ctx context.Context, c *Cart) error

	RemoveItem(ctx context.Context, userID uuid.UUID, item Item) error
	DeleteCart(ctx context.Context, userID uuid.UUID) error

	TTL(ctx context.Context, userID uuid.UUID) (time.Duration, error)
}
