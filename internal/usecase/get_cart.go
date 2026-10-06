package usecase

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/google/uuid"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) GetCart(ctx context.Context, input dto.GetCartInput) (dto.CartOutput, error) {
	output, err := u.cart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.cart: %w", err)
	}

	return output, nil
}

// cart — корзина пользователя: позиции по дате, затем по ID услуги, и секунды
// до истечения с округлением вверх (живая корзина не покажет 0).
func (u *UseCase) cart(ctx context.Context, userID uuid.UUID) (dto.CartOutput, error) {
	var output dto.CartOutput

	c, err := u.redis.GetCart(ctx, userID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCart: %w", err)
	}

	ttl, err := u.redis.GetCartTTL(ctx, userID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCartTTL: %w", err)
	}

	items := make([]cart.Item, 0, len(c.Items))
	for item := range c.Items {
		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b cart.Item) int {
		return cmp.Or(compareDates(a.Date, b.Date), cmp.Compare(a.ServiceID, b.ServiceID))
	})

	output.Items = make([]dto.CartItem, 0, len(items))
	for _, item := range items {
		output.Items = append(output.Items, dto.CartItem{ServiceID: item.ServiceID, Date: item.Date.String()})
	}

	output.ExpiresIn = int(math.Ceil(ttl.Seconds()))

	return output, nil
}

func compareDates(a, b service.Date) int {
	return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Month, b.Month), cmp.Compare(a.Day, b.Day))
}
