package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_ClearCart_Success(t *testing.T) {
	userID := uuid.New()

	redis := new(mocks.Redis)
	redis.On("DeleteCart", Any, userID).Return(nil)
	defer redis.AssertCalled(t, "DeleteCart", Any, userID)

	u := usecase.New(nil, redis, nil, nil)

	{
		err := u.ClearCart(context.Background(), dto.ClearCartInput{UserID: userID})
		require.NoError(t, err)
	}
}

func Test_ClearCart_Error(t *testing.T) {
	userID := uuid.New()
	errRedis := errors.New("redis down")

	redis := new(mocks.Redis)
	redis.On("DeleteCart", Any, userID).Return(errRedis)

	u := usecase.New(nil, redis, nil, nil)

	{
		err := u.ClearCart(context.Background(), dto.ClearCartInput{UserID: userID})
		require.ErrorIs(t, err, errRedis)
	}
}
