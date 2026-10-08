package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) RemoveCartItem(ctx context.Context, input dto.RemoveCartItemInput) (dto.CartOutput, error) {
	var output dto.CartOutput

	date, err := service.ParseDate(input.Date)
	if err != nil {
		return output, fmt.Errorf("service.ParseDate: %w", err)
	}

	err = u.redis.RemoveCartItem(ctx, input.UserID, cart.Item{ServiceID: input.ServiceID, Date: date})
	if err != nil {
		return output, fmt.Errorf("u.redis.RemoveCartItem: %w", err)
	}

	c, err := u.redis.GetCart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCart: %w", err)
	}

	return toCart(c), nil
}
