package usecase

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) GetCart(ctx context.Context, input dto.GetCartInput) (dto.CartOutput, error) {
	var output dto.CartOutput

	c, err := u.redis.GetCart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCart: %w", err)
	}

	return toCart(c), nil
}

// toCart — позиции по дате, затем по ID услуги, и секунды до истечения
// с округлением вверх (живая корзина не покажет 0). Дата YYYY-MM-DD
// сравнивается как строка так же, как дата.
func toCart(c *cart.Cart) dto.CartOutput {
	output := dto.CartOutput{
		Items:     make([]dto.CartItem, 0, len(c.Items)),
		ExpiresIn: int(math.Ceil(c.TTL.Seconds())),
	}

	for item := range c.Items {
		output.Items = append(output.Items, dto.CartItem{ServiceID: item.ServiceID, Date: item.Date.String()})
	}

	slices.SortFunc(output.Items, func(a, b dto.CartItem) int {
		return cmp.Or(cmp.Compare(a.Date, b.Date), cmp.Compare(a.ServiceID, b.ServiceID))
	})

	return output
}

func compareDates(a, b service.Date) int {
	return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Month, b.Month), cmp.Compare(a.Day, b.Day))
}
