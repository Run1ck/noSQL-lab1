package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/user"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_EnsureAdmin(t *testing.T) {
	isAdmin := mock.MatchedBy(func(u *user.User) bool { return u.Login == "admin" && u.Role == user.RoleAdmin })

	t.Run("created", func(t *testing.T) {
		postgres := new(mocks.Postgres)
		postgres.On("CreateUser", Any, isAdmin).Return(nil)
		u := usecase.New(postgres, nil, nil, nil)

		actual, err := u.EnsureAdmin(context.Background(), dto.EnsureAdminInput{Login: "admin", Password: "password123"})
		require.NoError(t, err)
		require.True(t, actual.Created)
	})

	t.Run("already exists", func(t *testing.T) {
		postgres := new(mocks.Postgres)
		postgres.On("CreateUser", Any, isAdmin).Return(user.ErrLoginTaken)
		u := usecase.New(postgres, nil, nil, nil)

		actual, err := u.EnsureAdmin(context.Background(), dto.EnsureAdminInput{Login: "admin", Password: "password123"})
		require.NoError(t, err)
		require.False(t, actual.Created)
	})

	t.Run("no password", func(t *testing.T) {
		postgres := new(mocks.Postgres)
		u := usecase.New(postgres, nil, nil, nil)

		actual, err := u.EnsureAdmin(context.Background(), dto.EnsureAdminInput{Login: "admin"})
		require.NoError(t, err)
		require.False(t, actual.Created)
		postgres.AssertNotCalled(t, "CreateUser", Any, Any)
	})
}
