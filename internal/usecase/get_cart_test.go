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
	userID := uuid.New()
	c := newCart(userID,
		cart.Item{ServiceID: "room-101", Date: date("2026-10-06")},
		cart.Item{ServiceID: "room-101", Date: date("2026-10-05")},
		cart.Item{ServiceID: "lab-1", Date: date("2026-10-06")},
	)

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(c, nil)
	redis.On("GetCartTTL", Any, userID).Return(1799500*time.Millisecond, nil)
	defer redis.AssertCalled(t, "GetCart", Any, userID)

	u := usecase.New(nil, redis, nil, nil)

	{
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
	userID := uuid.New()

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID), nil)
	redis.On("GetCartTTL", Any, userID).Return(time.Duration(0), nil)

	u := usecase.New(nil, redis, nil, nil)

	{
		actual, err := u.GetCart(context.Background(), dto.GetCartInput{UserID: userID})
		require.NoError(t, err)
		require.Equal(t, dto.CartOutput{Items: []dto.CartItem{}}, actual)
	}
}

func Test_GetCart_Error(t *testing.T) {
	userID := uuid.New()
	errRedis := errors.New("redis down")

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(nil, errRedis)

	u := usecase.New(nil, redis, nil, nil)

	{
		_, err := u.GetCart(context.Background(), dto.GetCartInput{UserID: userID})
		require.ErrorIs(t, err, errRedis)
	}
}
