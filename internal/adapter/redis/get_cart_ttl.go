package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Redis) GetCartTTL(ctx context.Context, userID uuid.UUID) (time.Duration, error) {
	ttl, err := r.redis.PTTL(ctx, cartKey(userID)).Result()
	if err != nil {
		return 0, fmt.Errorf("r.redis.PTTL: %w", err)
	}

	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}
