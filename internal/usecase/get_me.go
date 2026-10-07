package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) GetMe(ctx context.Context, input dto.GetMeInput) (dto.User, error) {
	var output dto.User

	usr, err := u.postgres.GetUserByID(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetUserByID: %w", err)
	}

	return toUser(usr), nil
}
