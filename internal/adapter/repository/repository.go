package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/usecase"
)

const (
	servicesKey    = "cache:services"
	schedulePrefix = "cache:schedule:"
)

type Repository struct {
	usecase.Postgres
	redis *redis.Client
	ttl   time.Duration
}

var _ usecase.Postgres = (*Repository)(nil)

func New(client *redis.Client, postgres usecase.Postgres, ttl time.Duration) *Repository {
	return &Repository{
		Postgres: postgres,
		redis:    client,
		ttl:      ttl,
	}
}

func scheduleKey(d service.Date) string {
	return schedulePrefix + d.String()
}

func (r *Repository) GetServices(ctx context.Context) ([]*service.Service, error) {
	var services []*service.Service
	if r.load(ctx, servicesKey, &services) {
		return services, nil
	}

	services, err := r.Postgres.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	r.store(ctx, servicesKey, services)

	return services, nil
}

func (r *Repository) GetService(ctx context.Context, id string) (*service.Service, error) {
	services, err := r.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range services {
		if s.ID == id {
			return s, nil
		}
	}

	return r.Postgres.GetService(ctx, id)
}

func (r *Repository) SaveService(ctx context.Context, s *service.Service) error {
	err := r.Postgres.SaveService(ctx, s)
	if err != nil {
		return err
	}

	r.invalidate(ctx, servicesKey)

	return nil
}

func (r *Repository) GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error) {
	key := scheduleKey(date)

	var bookings []booking.Booking
	if r.load(ctx, key, &bookings) {
		return bookings, nil
	}

	bookings, err := r.Postgres.GetBookings(ctx, date)
	if err != nil {
		return nil, err
	}

	r.store(ctx, key, bookings)

	return bookings, nil
}

func (r *Repository) Approve(ctx context.Context, req *request.Request) error {
	err := r.Postgres.Approve(ctx, req)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(req.Items))
	seen := make(map[service.Date]struct{}, len(req.Items))
	for _, item := range req.Items {
		if _, ok := seen[item.Date]; ok {
			continue
		}
		seen[item.Date] = struct{}{}
		keys = append(keys, scheduleKey(item.Date))
	}

	r.invalidate(ctx, keys...)

	return nil
}

func (r *Repository) load(ctx context.Context, key string, dst any) bool {
	data, err := r.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false
	}
	if err != nil {
		slog.Warn("cache read failed, falling back to postgres", "key", key, "err", err)

		return false
	}

	err = json.Unmarshal(data, dst)
	if err != nil {
		slog.Warn("cache entry is corrupted", "key", key, "err", err)

		return false
	}

	return true
}

func (r *Repository) store(ctx context.Context, key string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Warn("cache encode failed", "key", key, "err", err)

		return
	}

	err = r.redis.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		slog.Warn("cache write failed", "key", key, "err", err)
	}
}

func (r *Repository) invalidate(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}

	err := r.redis.Del(ctx, keys...).Err()
	if err != nil {
		slog.Warn("cache invalidation failed, entries stay until TTL", "keys", keys, "err", err)
	}
}
