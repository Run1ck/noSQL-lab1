package booking

import (
	"booking/internal/domain/service"
	"context"
)

type Booking struct {
	ServiceID string
	Date      service.Date
	RequestID int64
}

type Repository interface {
	ListByDate(ctx context.Context, d service.Date) ([]Booking, error)
}
