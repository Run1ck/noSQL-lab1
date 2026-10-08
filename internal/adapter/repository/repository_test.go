package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/adapter/repository"
	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/usecase/mocks"
)

const (
	Any = mock.Anything
	ttl = time.Minute
)

func date(s string) service.Date {
	d, err := service.ParseDate(s)
	if err != nil {
		panic(err)
	}

	return d
}

// newRepository — кэш на miniredis поверх мока Postgres.
func newRepository(t *testing.T) (*repository.Repository, *mocks.Postgres, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	postgres := new(mocks.Postgres)

	return repository.New(client, postgres, ttl), postgres, mr
}

func Test_GetServices_CacheHit(t *testing.T) {
	// Данные для поведения
	services := []*service.Service{
		{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
		{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
	}

	// Настраиваем поведение Postgres
	repo, postgres, mr := newRepository(t)
	postgres.On("GetServices", Any).Return(services, nil)

	{ // Сам тест: первый вызов — из Postgres, второй — из кэша
		for range 2 {
			actual, err := repo.GetServices(context.Background())
			require.NoError(t, err)
			require.Equal(t, services, actual)
		}

		postgres.AssertNumberOfCalls(t, "GetServices", 1)
		require.Equal(t, ttl, mr.TTL("cache:services"))
	}
}

func Test_GetService_FromCachedList(t *testing.T) {
	// Данные для поведения
	room := &service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true}

	// Настраиваем поведение Postgres
	repo, postgres, _ := newRepository(t)
	postgres.On("GetServices", Any).Return([]*service.Service{room}, nil)

	{ // Сам тест: обе услуги берутся из одного закэшированного списка
		actual, err := repo.GetService(context.Background(), "room-101")
		require.NoError(t, err)
		require.Equal(t, room, actual)

		_, err = repo.GetService(context.Background(), "nope")
		require.ErrorIs(t, err, service.ErrNotFound)

		postgres.AssertNumberOfCalls(t, "GetServices", 1)
		postgres.AssertNotCalled(t, "GetService", Any, Any)
	}
}

func Test_GetBookings_CachesFreeDay(t *testing.T) {
	// Данные для поведения
	busy, free := date("2026-10-05"), date("2026-10-06")
	bookings := []booking.Booking{{ServiceID: "room-101", Date: busy, RequestID: 1}}

	// Настраиваем поведение Postgres
	repo, postgres, mr := newRepository(t)
	postgres.On("GetBookings", Any, busy).Return(bookings, nil)
	postgres.On("GetBookings", Any, free).Return([]booking.Booking{}, nil)

	{ // Сам тест: и занятый, и свободный день после первого раза — из кэша
		for range 2 {
			actual, err := repo.GetBookings(context.Background(), busy)
			require.NoError(t, err)
			require.Equal(t, bookings, actual)

			actual, err = repo.GetBookings(context.Background(), free)
			require.NoError(t, err)
			require.Empty(t, actual)
		}

		postgres.AssertNumberOfCalls(t, "GetBookings", 2)
		require.True(t, mr.Exists("cache:schedule:2026-10-06"))
	}
}

func Test_SaveService_DropsServicesCache(t *testing.T) {
	// Данные для поведения
	room := &service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true}

	// Настраиваем поведение Postgres
	repo, postgres, mr := newRepository(t)
	postgres.On("GetServices", Any).Return([]*service.Service{room}, nil)
	postgres.On("SaveService", Any, room).Return(nil)

	{ // Сам тест: после сохранения список читается из Postgres заново
		_, err := repo.GetServices(context.Background())
		require.NoError(t, err)
		require.True(t, mr.Exists("cache:services"))

		require.NoError(t, repo.SaveService(context.Background(), room))
		require.False(t, mr.Exists("cache:services"))

		_, err = repo.GetServices(context.Background())
		require.NoError(t, err)
		postgres.AssertNumberOfCalls(t, "GetServices", 2)
	}
}

func Test_Approve_DropsScheduleOfRequestDays(t *testing.T) {
	// Данные для поведения
	req := &request.Request{ID: 1, Items: []request.Item{
		{ServiceID: "room-101", Date: date("2026-10-05")},
		{ServiceID: "lab-1", Date: date("2026-10-06")},
	}}

	// Настраиваем поведение Postgres
	repo, postgres, mr := newRepository(t)
	postgres.On("Approve", Any, req).Return(nil)

	for _, key := range []string{"cache:schedule:2026-10-05", "cache:schedule:2026-10-06", "cache:schedule:2026-10-07"} {
		require.NoError(t, mr.Set(key, "[]"))
	}

	{ // Сам тест: сброшены только дни заявки
		require.NoError(t, repo.Approve(context.Background(), req))
		require.False(t, mr.Exists("cache:schedule:2026-10-05"))
		require.False(t, mr.Exists("cache:schedule:2026-10-06"))
		require.True(t, mr.Exists("cache:schedule:2026-10-07"))
	}
}

func Test_Approve_ErrorKeepsCache(t *testing.T) {
	// Данные для поведения
	req := &request.Request{ID: 1, Items: []request.Item{{ServiceID: "room-101", Date: date("2026-10-05")}}}

	// Настраиваем поведение Postgres: день уже занят
	repo, postgres, mr := newRepository(t)
	postgres.On("Approve", Any, req).Return(request.ErrSlotBooked)
	require.NoError(t, mr.Set("cache:schedule:2026-10-05", "[]"))

	{ // Сам тест
		err := repo.Approve(context.Background(), req)
		require.ErrorIs(t, err, request.ErrSlotBooked)
		require.True(t, mr.Exists("cache:schedule:2026-10-05"))
	}
}

func Test_RedisDown_FallsBackToPostgres(t *testing.T) {
	// Данные для поведения
	services := []*service.Service{{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true}}

	// Настраиваем поведение Postgres, Redis недоступен
	repo, postgres, mr := newRepository(t)
	postgres.On("GetServices", Any).Return(services, nil)
	mr.SetError("ERR boom")

	{ // Сам тест: кэш не роняет запрос
		actual, err := repo.GetServices(context.Background())
		require.NoError(t, err)
		require.Equal(t, services, actual)
	}
}

func Test_PostgresError_NotCached(t *testing.T) {
	// Данные для поведения
	errDB := errors.New("db down")

	// Настраиваем поведение Postgres
	repo, postgres, mr := newRepository(t)
	postgres.On("GetServices", Any).Return(nil, errDB)

	{ // Сам тест
		_, err := repo.GetServices(context.Background())
		require.ErrorIs(t, err, errDB)
		require.False(t, mr.Exists("cache:services"))
	}
}

func Test_PassThrough(t *testing.T) {
	// Данные для поведения
	req := &request.Request{ID: 7}

	// Настраиваем поведение Postgres
	repo, postgres, _ := newRepository(t)
	postgres.On("GetRequest", Any, int64(7)).Return(req, nil)

	{ // Сам тест: остальные методы идут прямо в Postgres
		actual, err := repo.GetRequest(context.Background(), 7)
		require.NoError(t, err)
		require.Equal(t, req, actual)
	}
}
