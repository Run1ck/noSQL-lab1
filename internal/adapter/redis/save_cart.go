package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
)

// SaveCart перезаписывает корзину целиком: DEL, SADD и PEXPIRE в одной
// транзакции MULTI/EXEC, чтобы ключ не остался без TTL. Пустая корзина удаляется.
func (r *Redis) SaveCart(ctx context.Context, c *cart.Cart) error {
	if c.IsEmpty() {
		return r.DeleteCart(ctx, c.UserID)
	}

	members := make([]any, 0, len(c.Items))

	for item := range c.Items {
		// Битая позиция не прочитается обратно, и GetCart будет падать до конца TTL.
		if item.ServiceID == "" || !item.Date.IsValid() {
			return fmt.Errorf("invalid cart item %+v", item)
		}

		members = append(members, encodeItem(item))
	}

	key := cartKey(c.UserID)

	_, err := r.redis.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, key)
		p.SAdd(ctx, key, members...)
		p.PExpire(ctx, key, r.cartTTL)

		return nil
	})
	if err != nil {
		return fmt.Errorf("r.redis.TxPipelined: %w", err)
	}

	return nil
}
