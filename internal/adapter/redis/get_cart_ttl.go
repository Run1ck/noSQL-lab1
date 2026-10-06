package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GetCartTTL — сколько корзине осталось жить; 0 — корзины нет.
func (r *Redis) GetCartTTL(ctx context.Context, userID uuid.UUID) (time.Duration, error) {
	ttl, err := r.redis.PTTL(ctx, cartKey(userID)).Result()
	if err != nil {
		return 0, fmt.Errorf("r.redis.PTTL: %w", err)
	}

	// -2 — ключа нет, -1 — ключ без TTL: такой корзину не пишем.
	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}
