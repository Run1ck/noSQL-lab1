package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"booking/internal/auth"
	"booking/internal/domain/user"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_Login_Success(t *testing.T) {
	usr, err := user.New("ivanov", "password123", user.RoleAdmin)
	require.NoError(t, err)

	postgres := new(mocks.Postgres)
	postgres.On("GetUserByLogin", Any, "ivanov").Return(usr, nil)

	tokens := newTokens()
	u := usecase.New(postgres, nil, tokens, nil)

	actual, err := u.Login(context.Background(), dto.LoginInput{Login: " IVANOV ", Password: "password123"})
	require.NoError(t, err)
	require.Equal(t, dto.User{ID: usr.ID, Login: "ivanov", Role: "admin"}, actual.User)
	require.WithinDuration(t, time.Now().Add(time.Hour), actual.ExpiresAt, time.Minute)

	p, err := tokens.Parse(actual.Token)
	require.NoError(t, err)
	require.Equal(t, auth.Principal{UserID: usr.ID, Role: user.RoleAdmin}, p)
}

func Test_Login_InvalidCredentials(t *testing.T) {
	usr, err := user.New("ivanov", "password123", user.RoleUser)
	require.NoError(t, err)

	postgres := new(mocks.Postgres)
	postgres.On("GetUserByLogin", Any, "ivanov").Return(usr, nil)
	postgres.On("GetUserByLogin", Any, "nobody").Return(nil, user.ErrNotFound)

	u := usecase.New(postgres, nil, newTokens(), nil)

	for _, input := range []dto.LoginInput{
		{Login: "ivanov", Password: "wrong-password"},
		{Login: "nobody", Password: "password123"},
	} {
		_, err := u.Login(context.Background(), input)
		require.ErrorIs(t, err, user.ErrInvalidCredentials, input.Login)
	}
}
