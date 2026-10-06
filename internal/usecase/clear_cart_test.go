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
	// Данные для поведения
	userID := uuid.New()

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("DeleteCart", Any, userID).Return(nil)
	defer redis.AssertCalled(t, "DeleteCart", Any, userID)

	// Собираем UseCase
	u := usecase.New(nil, redis)

	{ // Сам тест
		err := u.ClearCart(context.Background(), dto.ClearCartInput{UserID: userID})
		require.NoError(t, err)
	}
}

func Test_ClearCart_Error(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()
	errRedis := errors.New("redis down")

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("DeleteCart", Any, userID).Return(errRedis)

	// Собираем UseCase
	u := usecase.New(nil, redis)

	{ // Сам тест
		err := u.ClearCart(context.Background(), dto.ClearCartInput{UserID: userID})
		require.ErrorIs(t, err, errRedis)
	}
}
