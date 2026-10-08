package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/adapter/repository"
	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/internal/usecase/mocks"
)

const (
	Any      = mock.Anything
	cacheTTL = time.Minute
)

func newRepo(t *testing.T) (*repository.Repository, *mocks.Postgres, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), DialTimeout: 100 * time.Millisecond, MaxRetries: -1, DialerRetries: 1})
	t.Cleanup(func() { _ = client.Close() })

	postgres := new(mocks.Postgres)

	return repository.New(client, postgres, cacheTTL), postgres, mr
}

func date(s string) service.Date {
	d, err := service.ParseDate(s)
	if err != nil {
		panic(err)
	}

	return d
}

func services() []*service.Service {
	return []*service.Service{
		{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
		{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
	}
}

func Test_GetServices_CachesWithTTL(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	postgres.On("GetServices", Any).Return(services(), nil).Once()
	ctx := context.Background()

	for range 3 {
		actual, err := repo.GetServices(ctx)
		require.NoError(t, err)
		require.Equal(t, services(), actual)
	}

	postgres.AssertNumberOfCalls(t, "GetServices", 1)
	require.Equal(t, cacheTTL, mr.TTL("cache:services"))

	mr.FastForward(cacheTTL)
	postgres.On("GetServices", Any).Return(services(), nil).Once()

	_, err := repo.GetServices(ctx)
	require.NoError(t, err)
	postgres.AssertNumberOfCalls(t, "GetServices", 2)
}

func Test_GetService_FromCache(t *testing.T) {
	repo, postgres, _ := newRepo(t)
	postgres.On("GetServices", Any).Return(services(), nil).Once()
	postgres.On("GetService", Any, "garage").Return(nil, service.ErrNotFound)
	ctx := context.Background()

	actual, err := repo.GetService(ctx, "room-101")
	require.NoError(t, err)
	require.Equal(t, "Аудитория 101", actual.Name)

	_, err = repo.GetService(ctx, "lab-1")
	require.NoError(t, err)

	_, err = repo.GetService(ctx, "garage")
	require.ErrorIs(t, err, service.ErrNotFound)

	postgres.AssertNumberOfCalls(t, "GetServices", 1)
}

func Test_SaveService_Invalidates(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	postgres.On("GetServices", Any).Return(services(), nil)
	postgres.On("SaveService", Any, Any).Return(service.ErrInvalid).Once()
	postgres.On("SaveService", Any, Any).Return(nil).Once()
	ctx := context.Background()

	_, err := repo.GetServices(ctx)
	require.NoError(t, err)

	err = repo.SaveService(ctx, &service.Service{ID: "room-101"})
	require.ErrorIs(t, err, service.ErrInvalid)
	require.True(t, mr.Exists("cache:services"))

	err = repo.SaveService(ctx, &service.Service{ID: "room-101"})
	require.NoError(t, err)
	require.False(t, mr.Exists("cache:services"))
}

func Test_GetBookings_CachesPerDay(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	d1, d2 := date("2026-10-05"), date("2026-10-06")
	booked := []booking.Booking{{ServiceID: "room-101", Date: d1, RequestID: 42}}
	postgres.On("GetBookings", Any, d1).Return(booked, nil).Once()
	postgres.On("GetBookings", Any, d2).Return([]booking.Booking{}, nil).Once()
	ctx := context.Background()

	for range 2 {
		actual, err := repo.GetBookings(ctx, d1)
		require.NoError(t, err)
		require.Equal(t, booked, actual)

		actual, err = repo.GetBookings(ctx, d2)
		require.NoError(t, err)
		require.Empty(t, actual)
	}

	postgres.AssertNumberOfCalls(t, "GetBookings", 2)
	require.True(t, mr.Exists("cache:schedule:2026-10-05"))
	require.True(t, mr.Exists("cache:schedule:2026-10-06"))
}

func Test_Approve_InvalidatesRequestDays(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	d1, d2, other := date("2026-10-05"), date("2026-10-06"), date("2026-10-07")
	postgres.On("GetBookings", Any, Any).Return([]booking.Booking{}, nil)
	postgres.On("Approve", Any, Any).Return(request.ErrSlotBooked).Once()
	postgres.On("Approve", Any, Any).Return(nil).Once()
	ctx := context.Background()

	for _, d := range []service.Date{d1, d2, other} {
		_, err := repo.GetBookings(ctx, d)
		require.NoError(t, err)
	}

	req := &request.Request{ID: 42, Items: []request.Item{
		{ServiceID: "room-101", Date: d1},
		{ServiceID: "lab-1", Date: d1},
		{ServiceID: "room-101", Date: d2},
	}}

	err := repo.Approve(ctx, req)
	require.ErrorIs(t, err, request.ErrSlotBooked)
	require.True(t, mr.Exists("cache:schedule:2026-10-05"))

	err = repo.Approve(ctx, req)
	require.NoError(t, err)
	require.False(t, mr.Exists("cache:schedule:2026-10-05"))
	require.False(t, mr.Exists("cache:schedule:2026-10-06"))
	require.True(t, mr.Exists("cache:schedule:2026-10-07"))
}

func Test_FailOpen_WhenRedisDown(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	postgres.On("GetServices", Any).Return(services(), nil)
	postgres.On("GetBookings", Any, Any).Return([]booking.Booking{}, nil)
	postgres.On("Approve", Any, Any).Return(nil)
	mr.Close()
	ctx := context.Background()

	actual, err := repo.GetServices(ctx)
	require.NoError(t, err)
	require.Len(t, actual, 2)

	_, err = repo.GetBookings(ctx, date("2026-10-05"))
	require.NoError(t, err)

	err = repo.Approve(ctx, &request.Request{Items: []request.Item{{ServiceID: "room-101", Date: date("2026-10-05")}}})
	require.NoError(t, err)
}

func Test_CorruptedEntry_FallsBack(t *testing.T) {
	repo, postgres, mr := newRepo(t)
	postgres.On("GetServices", Any).Return(services(), nil).Once()
	require.NoError(t, mr.Set("cache:services", "{not json"))

	actual, err := repo.GetServices(context.Background())
	require.NoError(t, err)
	require.Equal(t, services(), actual)
}

func Test_PassesThroughUncachedMethods(t *testing.T) {
	repo, postgres, _ := newRepo(t)
	id := uuid.New()
	postgres.On("GetUserByID", Any, id).Return(&user.User{ID: id}, nil)

	actual, err := repo.GetUserByID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, id, actual.ID)
}
