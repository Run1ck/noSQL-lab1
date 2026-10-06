package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) ClearCart(ctx context.Context, input dto.ClearCartInput) error {
	err := u.redis.DeleteCart(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("u.redis.DeleteCart: %w", err)
	}

	return nil
}
