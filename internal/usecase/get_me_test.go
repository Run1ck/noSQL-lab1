package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/user"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_GetMe(t *testing.T) {
	id := uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetUserByID", Any, id).Return(&user.User{ID: id, Login: "ivanov", Role: user.RoleUser}, nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.GetMe(context.Background(), dto.GetMeInput{UserID: id})
	require.NoError(t, err)
	require.Equal(t, dto.User{ID: id, Login: "ivanov", Role: "user"}, actual)
}
