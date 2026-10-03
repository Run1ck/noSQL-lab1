package request

import (
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ServiceID string
	Date      service.Date
}

type Request struct {
	ID          int64
	UserID      uuid.UUID
	Items       []Item
	Status      Status
	Comment     string
	CreatedAt   time.Time
	ProcessedAt *time.Time
	// ProcessedBy — админ, обработавший заявку, или сам заявитель при отмене.
	ProcessedBy *uuid.UUID
}

func NewFromCart(c *cart.Cart) (*Request, error) {
	if c == nil || c.IsEmpty() {
		return nil, ErrEmptyCart
	}

	items := make([]Item, 0, len(c.Items))
	for i := range c.Items {
		items = append(items, Item{
			ServiceID: i.ServiceID,
			Date:      i.Date,
		})
	}
	return &Request{
		UserID:    c.UserID,
		Items:     items,
		Status:    StatusNew,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Request) process(to Status, by uuid.UUID, comment string) error {
	if r.Status.IsFinal() {
		return ErrAlreadyProcessed
	}
	now := time.Now()
	r.Status = to
	r.Comment = strings.TrimSpace(comment)
	r.ProcessedAt = &now
	r.ProcessedBy = &by
	return nil
}

func (r *Request) Approve(adminID uuid.UUID, comment string) error {
	return r.process(StatusApproved, adminID, comment)
}

func (r *Request) Reject(adminID uuid.UUID, comment string) error {
	return r.process(StatusRejected, adminID, comment)
}

func (r *Request) Cancel(userID uuid.UUID) error {
	if userID != r.UserID {
		return ErrNotOwner
	}
	return r.process(StatusCancelled, userID, "")
}
