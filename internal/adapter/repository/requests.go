package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/usecase"
)

const schedulePrefix = "cache:schedule:"

type Requests struct {
	usecase.Requests
	cache
}

var _ usecase.Requests = (*Requests)(nil)

func NewRequests(client *redis.Client, requests usecase.Requests, ttl time.Duration) *Requests {
	return &Requests{
		Requests: requests,
		cache:    cache{redis: client, ttl: ttl},
	}
}

func scheduleKey(d service.Date) string {
	return schedulePrefix + d.String()
}

func (r *Requests) GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error) {
	key := scheduleKey(date)

	var bookings []booking.Booking
	if r.load(ctx, key, &bookings) {
		return bookings, nil
	}

	bookings, err := r.Requests.GetBookings(ctx, date)
	if err != nil {
		return nil, err
	}

	r.store(ctx, key, bookings)

	return bookings, nil
}

func (r *Requests) Approve(ctx context.Context, req *request.Request) error {
	err := r.Requests.Approve(ctx, req)
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
