package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
)

func (r *Redis) GetCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	members, err := r.redis.SMembers(ctx, cartKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("r.redis.SMembers: %w", err)
	}

	c := &cart.Cart{
		UserID: userID,
		Items:  make(map[cart.Item]struct{}, len(members)),
	}

	for _, m := range members {
		item, err := decodeItem(m)
		if err != nil {
			return nil, fmt.Errorf("decodeItem: %w", err)
		}

		c.Items[item] = struct{}{}
	}

	return c, nil
}

func decodeItem(s string) (cart.Item, error) {
	i := strings.LastIndexByte(s, '|')
	if i < 0 {
		return cart.Item{}, fmt.Errorf("bad cart item %q", s)
	}

	date, err := service.ParseDate(s[i+1:])
	if err != nil {
		return cart.Item{}, fmt.Errorf("bad cart item %q: %v", s, err)
	}

	return cart.Item{ServiceID: s[:i], Date: date}, nil
}
