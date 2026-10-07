package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/cart"
	"booking/internal/domain/request"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
	"booking/pkg/ratelimit"
)

func Test_CreateRequest_Success(t *testing.T) {
	userID := uuid.New()
	d1, d2 := date("2026-10-06"), date("2026-10-05")
	c := newCart(userID, cart.Item{ServiceID: "room-101", Date: d1}, cart.Item{ServiceID: "lab-1", Date: d2})
	createdAt := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

	postgres := new(mocks.Postgres)
	postgres.On("CreateRequest", Any, mock.MatchedBy(func(r *request.Request) bool {
		return r.UserID == userID && r.Status == request.StatusNew && len(r.Items) == 2
	})).Run(func(args mock.Arguments) {
		r := args.Get(1).(*request.Request)
		r.ID = 42
		r.CreatedAt = createdAt
	}).Return(nil)

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(c, nil)
	redis.On("DeleteCart", Any, userID).Return(nil)

	ban := &fakeBan{d: ratelimit.Decision{Allowed: true}}
	u := usecase.New(postgres, redis, nil, ban)

	actual, err := u.CreateRequest(context.Background(), dto.CreateRequestInput{UserID: userID})
	require.NoError(t, err)
	require.Equal(t, dto.Request{
		ID:     42,
		UserID: userID,
		Items: []dto.RequestItem{
			{ServiceID: "lab-1", Date: "2026-10-05"},
			{ServiceID: "room-101", Date: "2026-10-06"},
		},
		Status:    "new",
		CreatedAt: createdAt,
	}, actual)
	require.Equal(t, 1, ban.calls)
	redis.AssertCalled(t, "DeleteCart", Any, userID)
}

func Test_CreateRequest_EmptyCart(t *testing.T) {
	userID := uuid.New()

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID), nil)

	ban := &fakeBan{d: ratelimit.Decision{Allowed: true}}
	u := usecase.New(new(mocks.Postgres), redis, nil, ban)

	_, err := u.CreateRequest(context.Background(), dto.CreateRequestInput{UserID: userID})
	require.ErrorIs(t, err, request.ErrEmptyCart)
	require.Zero(t, ban.calls)
}

func Test_CreateRequest_Banned(t *testing.T) {
	userID := uuid.New()

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID, cart.Item{ServiceID: "room-101", Date: date("2026-10-05")}), nil)

	postgres := new(mocks.Postgres)
	u := usecase.New(postgres, redis, nil, &fakeBan{d: ratelimit.Decision{RetryAfter: time.Minute}})

	_, err := u.CreateRequest(context.Background(), dto.CreateRequestInput{UserID: userID})

	var limited *ratelimit.LimitedError
	require.ErrorAs(t, err, &limited)
	require.True(t, limited.Banned)
	require.Equal(t, time.Minute, limited.RetryAfter)
	postgres.AssertNotCalled(t, "CreateRequest", Any, Any)
	redis.AssertNotCalled(t, "DeleteCart", Any, Any)
}

func Test_CreateRequest_FailOpen(t *testing.T) {
	userID := uuid.New()

	redis := new(mocks.Redis)
	redis.On("GetCart", Any, userID).Return(newCart(userID, cart.Item{ServiceID: "room-101", Date: date("2026-10-05")}), nil)
	redis.On("DeleteCart", Any, userID).Return(errors.New("redis down"))

	postgres := new(mocks.Postgres)
	postgres.On("CreateRequest", Any, Any).Return(nil)

	u := usecase.New(postgres, redis, nil, &fakeBan{err: errors.New("redis down")})

	_, err := u.CreateRequest(context.Background(), dto.CreateRequestInput{UserID: userID})
	require.NoError(t, err)
	postgres.AssertCalled(t, "CreateRequest", Any, Any)
}
