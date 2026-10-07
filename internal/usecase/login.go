package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"booking/internal/auth"
	"booking/internal/domain/user"
	"booking/internal/dto"
)

func (u *UseCase) Login(ctx context.Context, input dto.LoginInput) (dto.LoginOutput, error) {
	var output dto.LoginOutput

	usr, err := u.postgres.GetUserByLogin(ctx, strings.ToLower(strings.TrimSpace(input.Login)))
	if errors.Is(err, user.ErrNotFound) {
		return output, user.ErrInvalidCredentials
	}
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetUserByLogin: %w", err)
	}

	err = usr.CheckPassword(input.Password)
	if err != nil {
		return output, err
	}

	token, expiresAt, err := u.tokens.Issue(auth.Principal{UserID: usr.ID, Role: usr.Role})
	if err != nil {
		return output, fmt.Errorf("u.tokens.Issue: %w", err)
	}

	return dto.LoginOutput{Token: token, ExpiresAt: expiresAt, User: toUser(usr)}, nil
}
