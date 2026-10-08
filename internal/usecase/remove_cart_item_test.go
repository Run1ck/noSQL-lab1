package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_RemoveCartItem_Success(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()
	item := cart.Item{ServiceID: "room-101", Date: date("2026-10-06")}
	rest := newCart(userID, cart.Item{ServiceID: "lab-1", Date: date("2026-10-06")})
	rest.TTL = 30 * time.Minute

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("RemoveCartItem", Any, userID, item).Return(nil)
	redis.On("GetCart", Any, userID).Return(rest, nil)
	defer redis.AssertCalled(t, "RemoveCartItem", Any, userID, item)

	// Собираем UseCase
	u := usecase.New(nil, redis, nil, nil)

	{ // Сам тест
		input := dto.RemoveCartItemInput{UserID: userID, ServiceID: "room-101", Date: "2026-10-06"}
		output := dto.CartOutput{
			Items:     []dto.CartItem{{ServiceID: "lab-1", Date: "2026-10-06"}},
			ExpiresIn: 1800,
		}

		actual, err := u.RemoveCartItem(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_RemoveCartItem_NotFound(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("RemoveCartItem", Any, userID, Any).Return(cart.ErrItemNotFound)

	// Собираем UseCase
	u := usecase.New(nil, redis, nil, nil)

	{ // Сам тест
		input := dto.RemoveCartItemInput{UserID: userID, ServiceID: "room-101", Date: "2026-10-06"}

		_, err := u.RemoveCartItem(context.Background(), input)
		require.ErrorIs(t, err, cart.ErrItemNotFound)
	}
}

func Test_RemoveCartItem_InvalidDate(t *testing.T) {
	// Собираем UseCase: до Redis дело не доходит
	u := usecase.New(nil, nil, nil, nil)

	{ // Сам тест
		input := dto.RemoveCartItemInput{UserID: uuid.New(), ServiceID: "room-101", Date: "2026-02-30"}

		_, err := u.RemoveCartItem(context.Background(), input)
		require.ErrorIs(t, err, service.ErrInvalidDate)
	}
}
