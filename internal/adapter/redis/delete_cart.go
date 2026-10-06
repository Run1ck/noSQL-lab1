package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DeleteCart удаляет корзину; если её нет — не ошибка.
func (r *Redis) DeleteCart(ctx context.Context, userID uuid.UUID) error {
	err := r.redis.Del(ctx, cartKey(userID)).Err()
	if err != nil {
		return fmt.Errorf("r.redis.Del: %w", err)
	}

	return nil
}
