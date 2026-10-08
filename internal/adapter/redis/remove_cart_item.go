package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
)

var removeItemScript = redis.NewScript(`
if redis.call('SREM', KEYS[1], ARGV[1]) == 0 then
	return 0
end
redis.call('PEXPIRE', KEYS[1], ARGV[2])
return 1
`)

func (r *Redis) RemoveCartItem(ctx context.Context, userID uuid.UUID, item cart.Item) error {
	removed, err := removeItemScript.Run(ctx, r.redis,
		[]string{cartKey(userID)}, encodeItem(item), r.cartTTL.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("removeItemScript.Run: %w", err)
	}

	if removed == 0 {
		return cart.ErrItemNotFound
	}

	return nil
}
