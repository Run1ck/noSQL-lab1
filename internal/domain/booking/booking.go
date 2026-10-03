package booking

import (
	"booking/internal/domain/service"
	"context"
)

// Booking — подтверждённая бронь: услуга занята на весь день одобренной
// заявкой. Брони создаёт request.Repository.Approve в одной транзакции
// с одобрением, поэтому здесь только чтение.
type Booking struct {
	ServiceID string
	Date      service.Date
	RequestID int64
}

type Repository interface {
	// ListByDate возвращает брони на день; пустой срез — все услуги свободны.
	ListByDate(ctx context.Context, d service.Date) ([]Booking, error)
}
