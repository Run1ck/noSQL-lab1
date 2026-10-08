package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Redis) DeleteCart(ctx context.Context, userID uuid.UUID) error {
	err := r.redis.Del(ctx, cartKey(userID)).Err()
	if err != nil {
		return fmt.Errorf("r.redis.Del: %w", err)
	}

	return nil
}
