package firestore

import (
	"context"

	"booking/internal/domain/booking"
	"booking/internal/domain/service"
)

func (f *Firestore) GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error) {
	return []booking.Booking{}, nil
}
