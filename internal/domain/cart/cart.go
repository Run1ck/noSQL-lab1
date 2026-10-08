package cart

import (
	"time"

	"github.com/google/uuid"

	"booking/internal/domain/service"
)

type Item struct {
	ServiceID string
	Date      service.Date
}

// Cart — корзина пользователя; TTL — сколько ей осталось жить, 0 — корзины нет.
type Cart struct {
	UserID uuid.UUID
	Items  map[Item]struct{}
	TTL    time.Duration
}

func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}
