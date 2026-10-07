package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepo struct {
	q *sqlc.Queries
}

var _ booking.Repository = (*BookingRepo)(nil)

func NewBookingRepo(pool *pgxpool.Pool) *BookingRepo {
	return &BookingRepo{q: sqlc.New(pool)}
}

func toDomainBooking(b sqlc.Booking) booking.Booking {
	return booking.Booking{
		ServiceID: b.ServiceID,
		Date:      service.DateOf(b.Date),
		RequestID: b.RequestID,
	}
}

func (r *BookingRepo) ListByDate(ctx context.Context, d service.Date) ([]booking.Booking, error) {
	rows, err := r.q.GetBookingsByDate(ctx, dateToDB(d))
	if err != nil {
		return nil, err
	}
	result := make([]booking.Booking, 0, len(rows))
	for _, b := range rows {
		result = append(result, toDomainBooking(b))
	}
	return result, nil
}
