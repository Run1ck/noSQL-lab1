package cart

import (
	"booking/internal/domain/service"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ServiceID string
	Date      service.Date
}

type Cart struct {
	UserID    uuid.UUID
	Items     map[Item]struct{}
	UpdatedAt time.Time
}

func New(userID string) (*Cart, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUUID
	}
	return &Cart{
		UserID:    id,
		Items:     make(map[Item]struct{}),
		UpdatedAt: time.Now(),
	}, nil
}

func (c *Cart) AddItem(item Item) {
	c.Items[item] = struct{}{}
	c.touch()
}

func (c *Cart) RemoveItem(item Item) {
	delete(c.Items, item)
	c.touch()
}

func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

func (c *Cart) touch() {
	c.UpdatedAt = time.Now()
}
