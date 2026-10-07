package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/user"
	"booking/internal/dto"
)

func (u *UseCase) Register(ctx context.Context, input dto.RegisterInput) (dto.User, error) {
	var output dto.User

	usr, err := user.New(input.Login, input.Password, user.RoleUser)
	if err != nil {
		return output, fmt.Errorf("user.New: %w", err)
	}

	err = u.postgres.CreateUser(ctx, usr)
	if err != nil {
		return output, fmt.Errorf("u.postgres.CreateUser: %w", err)
	}

	return toUser(usr), nil
}

func toUser(usr *user.User) dto.User {
	return dto.User{ID: usr.ID, Login: usr.Login, Role: string(usr.Role)}
}
