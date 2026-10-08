package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
)

// AddCartItem добавляет позицию и продлевает TTL всей корзины: SADD и PEXPIRE
// в одной транзакции MULTI/EXEC, чтобы ключ не остался без TTL. Повтор позиции —
// не ошибка, только продлевает TTL.
func (r *Redis) AddCartItem(ctx context.Context, userID uuid.UUID, item cart.Item) error {
	key := cartKey(userID)

	_, err := r.redis.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.SAdd(ctx, key, encodeItem(item))
		p.PExpire(ctx, key, r.cartTTL)

		return nil
	})
	if err != nil {
		return fmt.Errorf("r.redis.TxPipelined: %w", err)
	}

	return nil
}
