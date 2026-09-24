package cart

import (
	"slices"
	"time"
)

type Item struct {
	ServiceID   string
	ServiceName string
	Qty         int
}

type Cart struct {
	SessionID string
	Items     []Item
	UpdatedAt time.Time
}

func New(sessionID string) (*Cart, error) {
	if sessionID == "" {
		return nil, ErrEmptySession
	}
	return &Cart{
		SessionID: sessionID,
		UpdatedAt: time.Now(),
	}, nil
}

func (c *Cart) AddItem(serviceID, serviceName string, qty int) error {
	if qty <= 0 {
		return ErrInvalidQty
	}

	for i := range c.Items {
		if c.Items[i].ServiceID == serviceID {
			c.Items[i].Qty += qty
			c.touch()
			return nil
		}
	}

	c.Items = append(c.Items, Item{
		ServiceID:   serviceID,
		ServiceName: serviceName,
		Qty:         qty,
	})
	c.touch()
	return nil
}

func (c *Cart) RemoveItem(serviceID string) error {
	for i := range c.Items {
		if c.Items[i].ServiceID == serviceID {
			c.Items = slices.Delete(c.Items, i, i+1)
			c.touch()
			return nil
		}
	}
	return ErrItemNotFound
}

func (c *Cart) IsEmpty() bool { return len(c.Items) == 0 }

func (c *Cart) touch() { c.UpdatedAt = time.Now() }
