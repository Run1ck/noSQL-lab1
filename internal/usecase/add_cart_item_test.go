package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/booking"
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

// day — дата со сдвигом от сегодня: «сегодня» use case берёт из time.Now().
func day(offset int) service.Date {
	return service.DateOf(time.Now().AddDate(0, 0, offset))
}

func Test_AddCartItem_Success(t *testing.T) {
	// Данные для поведения
	userID := uuid.New()
	today := day(0) // сегодня — можно
	item := cart.Item{ServiceID: "room-101", Date: today}

	// Настраиваем поведение Postgres: в этот день занята другая услуга
	postgres := new(mocks.Postgres)
	postgres.On("GetService", Any, "room-101").Return(&service.Service{ID: "room-101", Active: true}, nil)
	postgres.On("GetBookings", Any, today).Return([]booking.Booking{{ServiceID: "lab-1", Date: today}}, nil)

	// Настраиваем поведение Redis
	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID), nil).Once()
	redis.On("SaveCart", Any, mock.MatchedBy(func(c *cart.Cart) bool {
		_, ok := c.Items[item]

		return ok && len(c.Items) == 1
	})).Return(nil)
	redis.On("GetCart", Any, userID).Return(newCart(userID, item), nil).Once()
	redis.On("GetCartTTL", Any, userID).Return(30*time.Minute, nil)
	defer redis.AssertCalled(t, "SaveCart", Any, Any)

	// Собираем UseCase
	u := usecase.New(postgres, redis)

	{ // Сам тест
		input := dto.AddCartItemInput{UserID: userID, ServiceID: "room-101", Date: today.String()}
		output := dto.CartOutput{
			Items:     []dto.CartItem{{ServiceID: "room-101", Date: today.String()}},
			ExpiresIn: 1800,
		}

		actual, err := u.AddCartItem(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_AddCartItem_Rejected(t *testing.T) {
	tomorrow := day(1)

	for _, tc := range []struct {
		name    string
		input   dto.AddCartItemInput
		service *service.Service
		booked  []booking.Booking
		err     error
	}{
		{
			name:  "invalid date",
			input: dto.AddCartItemInput{ServiceID: "room-101", Date: "2026-02-30"},
			err:   service.ErrInvalidDate,
		},
		{
			name:  "unknown service",
			input: dto.AddCartItemInput{ServiceID: "nope", Date: tomorrow.String()},
			err:   service.ErrNotFound,
		},
		{
			name:    "inactive service",
			input:   dto.AddCartItemInput{ServiceID: "old-lab", Date: tomorrow.String()},
			service: &service.Service{ID: "old-lab", Active: false},
			err:     cart.ErrServiceUnavailable,
		},
		{
			name:    "inactive and past: availability first",
			input:   dto.AddCartItemInput{ServiceID: "old-lab", Date: day(-1).String()},
			service: &service.Service{ID: "old-lab", Active: false},
			err:     cart.ErrServiceUnavailable,
		},
		{
			name:    "past date",
			input:   dto.AddCartItemInput{ServiceID: "room-101", Date: day(-1).String()},
			service: &service.Service{ID: "room-101", Active: true},
			err:     cart.ErrPastDate,
		},
		{
			name:    "booked slot",
			input:   dto.AddCartItemInput{ServiceID: "room-101", Date: tomorrow.String()},
			service: &service.Service{ID: "room-101", Active: true},
			booked:  []booking.Booking{{ServiceID: "room-101", Date: tomorrow}},
			err:     cart.ErrSlotBooked,
		},
	} {
		// Настраиваем поведение Postgres
		postgres := new(mocks.Postgres)
		if tc.service != nil {
			postgres.On("GetService", Any, tc.input.ServiceID).Return(tc.service, nil)
		} else {
			postgres.On("GetService", Any, tc.input.ServiceID).Return(nil, service.ErrNotFound)
		}
		postgres.On("GetBookings", Any, Any).Return(tc.booked, nil)

		// Собираем UseCase: Redis без ожиданий — корзину трогать нельзя
		u := usecase.New(postgres, new(mocks.Redis))

		{ // Сам тест
			actual, err := u.AddCartItem(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err, tc.name)
			require.Equal(t, dto.CartOutput{}, actual, tc.name)
		}
	}
}
