package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
)

// GetCart отдаёт корзину пользователя вместе с остатком TTL; нет ключа — пустая
// корзина с TTL 0, не ошибка. SMEMBERS и PTTL идут в одной транзакции
// MULTI/EXEC: корзина не истечёт между чтением позиций и TTL.
func (r *Redis) GetCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	key := cartKey(userID)

	var (
		members *redis.StringSliceCmd
		ttl     *redis.DurationCmd
	)

	_, err := r.redis.TxPipelined(ctx, func(p redis.Pipeliner) error {
		members = p.SMembers(ctx, key)
		ttl = p.PTTL(ctx, key)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("r.redis.TxPipelined: %w", err)
	}

	c := &cart.Cart{
		UserID: userID,
		Items:  make(map[cart.Item]struct{}, len(members.Val())),
		// -2 — ключа нет, -1 — ключ без TTL: такой корзину не пишем.
		TTL: max(ttl.Val(), 0),
	}

	for _, m := range members.Val() {
		item, err := decodeItem(m)
		if err != nil {
			return nil, fmt.Errorf("decodeItem: %w", err)
		}

		c.Items[item] = struct{}{}
	}

	return c, nil
}

// decodeItem режет по последнему '|': в дате его нет, а в ID услуги может быть.
func decodeItem(s string) (cart.Item, error) {
	i := strings.LastIndexByte(s, '|')
	if i < 0 {
		return cart.Item{}, fmt.Errorf("bad cart item %q", s)
	}

	date, err := service.ParseDate(s[i+1:])
	if err != nil {
		// %v, а не %w: битые данные в Redis — это 500, а не 400 invalid_date.
		return cart.Item{}, fmt.Errorf("bad cart item %q: %v", s, err)
	}

	return cart.Item{ServiceID: s[:i], Date: date}, nil
}
