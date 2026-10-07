package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/request"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_GetRequest(t *testing.T) {
	owner, other := uuid.New(), uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, owner, request.StatusNew), nil)
	postgres.On("GetRequest", Any, int64(7)).Return(nil, request.ErrNotFound)

	u := usecase.New(postgres, nil, nil, nil)
	ctx := context.Background()

	actual, err := u.GetRequest(ctx, dto.GetRequestInput{UserID: owner, ID: 42})
	require.NoError(t, err)
	require.Equal(t, int64(42), actual.ID)

	_, err = u.GetRequest(ctx, dto.GetRequestInput{UserID: other, IsAdmin: true, ID: 42})
	require.NoError(t, err)

	_, err = u.GetRequest(ctx, dto.GetRequestInput{UserID: other, ID: 42})
	require.ErrorIs(t, err, request.ErrNotOwner)

	_, err = u.GetRequest(ctx, dto.GetRequestInput{UserID: owner, ID: 7})
	require.ErrorIs(t, err, request.ErrNotFound)
}
