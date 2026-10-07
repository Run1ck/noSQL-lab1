package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"context"
	"time"
)

type BookingRepo struct {
	q *sqlc.Queries
}

func toDomainBooking(b sqlc.Booking) booking.Booking {
	return booking.Booking{
		ServiceID: b.ServiceID,
		Date:      service.DateOf(b.Date),
		RequestID: b.RequestID,
	}
}

func (r *BookingRepo) ListByDate(ctx context.Context, d service.Date) ([]booking.Booking, error) {
	row, err := r.q.GetBookingsByDate(ctx, time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local))
	if err != nil {
		return nil, err
	}
	result := make([]booking.Booking, 0, len(row))
	for _, b := range row {
		result = append(result, toDomainBooking(b))
	}
	return result, nil
}
