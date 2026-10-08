package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
)

// GetServices — из кэша; на промахе — из Postgres с записью в кэш. Если Redis
// недоступен, данные отдаются из Postgres: кэш не должен ронять запрос.
func (r *Repository) GetServices(ctx context.Context) ([]*service.Service, error) {
	var services []*service.Service

	err := r.getCache(ctx, servicesKey, &services)
	if err == nil {
		return services, nil
	}

	if !errors.Is(err, errCacheMiss) {
		slog.Error("cache: GetServices: get cache", "err", err)
	}

	services, err = r.postgres.GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("r.postgres.GetServices: %w", err)
	}

	err = r.setCache(ctx, servicesKey, services)
	if err != nil {
		slog.Error("cache: GetServices: set cache", "err", err)
	}

	return services, nil
}

// GetService ищет услугу в закэшированном списке: отдельного ключа нет, и
// сбрасывать при SaveService нужно только cache:services.
func (r *Repository) GetService(ctx context.Context, id string) (*service.Service, error) {
	services, err := r.GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("r.GetServices: %w", err)
	}

	for _, s := range services {
		if s.ID == id {
			return s, nil
		}
	}

	return nil, service.ErrNotFound
}

// GetBookings — брони дня из кэша. Свободный день кэшируется тоже, иначе
// большинство дней расписания всегда были бы промахом.
func (r *Repository) GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error) {
	var bookings []booking.Booking

	key := scheduleKey(date)

	err := r.getCache(ctx, key, &bookings)
	if err == nil {
		return bookings, nil
	}

	if !errors.Is(err, errCacheMiss) {
		slog.Error("cache: GetBookings: get cache", "err", err, "date", date.String())
	}

	bookings, err = r.postgres.GetBookings(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("r.postgres.GetBookings: %w", err)
	}

	err = r.setCache(ctx, key, bookings)
	if err != nil {
		slog.Error("cache: GetBookings: set cache", "err", err, "date", date.String())
	}

	return bookings, nil
}

func (r *Repository) SaveService(ctx context.Context, s *service.Service) error {
	err := r.postgres.SaveService(ctx, s)
	if err != nil {
		return fmt.Errorf("r.postgres.SaveService: %w", err)
	}

	err = r.deleteCache(ctx, servicesKey)
	if err != nil {
		slog.Error("cache: SaveService: delete cache", "err", err, "serviceID", s.ID)
	}

	return nil
}

// Approve занимает дни заявки — кэш этих дней сбрасывается. Если DEL не
// прошёл, расписание отстанет не дольше чем на TTL кэша; двойную бронь всё
// равно отсекает первичный ключ bookings.
func (r *Repository) Approve(ctx context.Context, req *request.Request) error {
	err := r.postgres.Approve(ctx, req)
	if err != nil {
		return fmt.Errorf("r.postgres.Approve: %w", err)
	}

	keys := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		keys = append(keys, scheduleKey(item.Date))
	}

	if len(keys) == 0 {
		return nil
	}

	err = r.deleteCache(ctx, keys...)
	if err != nil {
		slog.Error("cache: Approve: delete cache", "err", err, "requestID", req.ID)
	}

	return nil
}

func (r *Repository) Reject(ctx context.Context, req *request.Request) error {
	return r.postgres.Reject(ctx, req)
}

func (r *Repository) Cancel(ctx context.Context, req *request.Request) error {
	return r.postgres.Cancel(ctx, req)
}

func (r *Repository) CreateRequest(ctx context.Context, req *request.Request) error {
	return r.postgres.CreateRequest(ctx, req)
}

func (r *Repository) GetRequest(ctx context.Context, id int64) (*request.Request, error) {
	return r.postgres.GetRequest(ctx, id)
}

func (r *Repository) GetRequests(ctx context.Context) ([]*request.Request, error) {
	return r.postgres.GetRequests(ctx)
}

func (r *Repository) GetRequestsByStatus(ctx context.Context, s request.Status) ([]*request.Request, error) {
	return r.postgres.GetRequestsByStatus(ctx, s)
}

func (r *Repository) GetRequestsByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error) {
	return r.postgres.GetRequestsByUser(ctx, userID)
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (*user.User, error) {
	return r.postgres.GetUserByLogin(ctx, login)
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return r.postgres.GetUserByID(ctx, id)
}

func (r *Repository) CreateUser(ctx context.Context, u *user.User) error {
	return r.postgres.CreateUser(ctx, u)
}
