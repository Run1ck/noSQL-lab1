package usecase

import (
	"context"
	"errors"
	"fmt"

	"booking/internal/domain/user"
	"booking/internal/dto"
)

func (u *UseCase) EnsureAdmin(ctx context.Context, input dto.EnsureAdminInput) (dto.EnsureAdminOutput, error) {
	var output dto.EnsureAdminOutput

	if input.Password == "" {
		return output, nil
	}

	admin, err := user.New(input.Login, input.Password, user.RoleAdmin)
	if err != nil {
		return output, fmt.Errorf("user.New: %w", err)
	}

	err = u.postgres.CreateUser(ctx, admin)
	if errors.Is(err, user.ErrLoginTaken) {
		return output, nil
	}
	if err != nil {
		return output, fmt.Errorf("u.postgres.CreateUser: %w", err)
	}

	output.Created = true

	return output, nil
}
