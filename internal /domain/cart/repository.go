package cart

import (
	"context"
	"time"
)

type Repository interface {
	Get(ctx context.Context, sessionID string) (*Cart, error)
	Save(ctx context.Context, c *Cart) error

	RemoveItem(ctx context.Context, sessionID, serviceID string) error
	DeleteSession(ctx context.Context, sessionID string) error

	TTL(ctx context.Context, sessionID string) (time.Duration, error)
}
