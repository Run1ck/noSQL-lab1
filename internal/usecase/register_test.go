package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"booking/internal/domain/user"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_Register_Success(t *testing.T) {
	var saved *user.User

	postgres := new(mocks.Postgres)
	postgres.On("CreateUser", Any, mock.AnythingOfType("*user.User")).
		Run(func(args mock.Arguments) { saved = args.Get(1).(*user.User) }).
		Return(nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.Register(context.Background(), dto.RegisterInput{Login: " Ivanov ", Password: "password123"})
	require.NoError(t, err)
	require.Equal(t, dto.User{ID: saved.ID, Login: "ivanov", Role: "user"}, actual)
	require.Equal(t, user.RoleUser, saved.Role)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(saved.PasswordHash), []byte("password123")))
}

func Test_Register_Rejected(t *testing.T) {
	t.Run("short password", func(t *testing.T) {
		u := usecase.New(new(mocks.Postgres), nil, nil, nil)

		_, err := u.Register(context.Background(), dto.RegisterInput{Login: "ivanov", Password: "short"})
		require.ErrorIs(t, err, user.ErrInvalidPassword)
	})

	t.Run("login taken", func(t *testing.T) {
		postgres := new(mocks.Postgres)
		postgres.On("CreateUser", Any, Any).Return(user.ErrLoginTaken)
		u := usecase.New(postgres, nil, nil, nil)

		_, err := u.Register(context.Background(), dto.RegisterInput{Login: "ivanov", Password: "password123"})
		require.ErrorIs(t, err, user.ErrLoginTaken)
	})
}
