package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
)

// removeItemScript удаляет позицию и продлевает TTL одной атомарной операцией.
//
//	KEYS[1] — ключ корзины, cart:{userID}
//	ARGV[1] — позиция, "serviceID|YYYY-MM-DD"
//	ARGV[2] — TTL корзины в миллисекундах
//
// Возвращает 1, если позиция удалена; 0, если её не было — тогда TTL не трогает.
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
