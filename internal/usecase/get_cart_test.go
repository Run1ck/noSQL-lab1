package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/cart"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func newCart(userID uuid.UUID, items ...cart.Item) *cart.Cart {
	c := &cart.Cart{UserID: userID, Items: make(map[cart.Item]struct{})}
	for _, item := range items {
		c.Items[item] = struct{}{}
	}

	return c
}

func Test_GetCart_Success(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()
	c := newCart(userID,
		cart.Item{ServiceID: "room-101", Date: date("2026-10-06")},
		cart.Item{ServiceID: "room-101", Date: date("2026-10-05")},
		cart.Item{ServiceID: "lab-1", Date: date("2026-10-06")},
	)
	c.TTL = 1799500 * time.Millisecond

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(c, nil)
	defer redis.AssertCalled(t, "GetCart", Any, userID)

	// Собираем UseCase
	u := usecase.New(nil, redis, nil, nil)

	{ // Сам тест: по дате, затем по ID услуги; секунды — с округлением вверх
		output := dto.CartOutput{
			Items: []dto.CartItem{
				{ServiceID: "room-101", Date: "2026-10-05"},
				{ServiceID: "lab-1", Date: "2026-10-06"},
				{ServiceID: "room-101", Date: "2026-10-06"},
			},
			ExpiresIn: 1800,
		}

		actual, err := u.GetCart(context.Background(), dto.GetCartInput{UserID: userID})
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_GetCart_Empty(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()

	// Настраиваем поведение Redis: корзины нет
	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID), nil)

	// Собираем UseCase
	u := usecase.New(nil, redis, nil, nil)

	{ // Сам тест: пустой срез, а не nil — в JSON будет [], а не null
		actual, err := u.GetCart(context.Background(), dto.GetCartInput{UserID: userID})
		require.NoError(t, err)
		require.Equal(t, dto.CartOutput{Items: []dto.CartItem{}}, actual)
	}
}

func Test_GetCart_Error(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()
	errRedis := errors.New("redis down")

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(nil, errRedis)

	// Собираем UseCase
	u := usecase.New(nil, redis, nil, nil)

	{ // Сам тест
		_, err := u.GetCart(context.Background(), dto.GetCartInput{UserID: userID})
		require.ErrorIs(t, err, errRedis)
	}
}
