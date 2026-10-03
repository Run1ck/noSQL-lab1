package request

import (
	"booking/internal/domain/cart"
	"strings"
	"time"
)

type Applicant struct {
	Name  string
	Email string
	Group string
}

func NewApplicant(name, email, group string) (Applicant, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" {
		return Applicant{}, ErrInvalidApplicant
	}
	return Applicant{
		Name:  name,
		Email: email,
		Group: strings.TrimSpace(group),
	}, nil
}

type Item struct {
	ServiceID   string
	ServiceName string
	Qty         int
}

type Request struct {
	ID          int64
	Applicant   Applicant
	Items       []Item
	Status      Status
	Comment     string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

func NewFromCart(id int64, applicant Applicant, c *cart.Cart) (*Request, error) {
	if c == nil || c.IsEmpty() {
		return nil, ErrEmptyCart
	}

	items := make([]Item, 0, len(c.Items))
	for _, i := range c.Items {
		items = append(items, Item{
			ServiceID:   i.ServiceID,
			ServiceName: i.ServiceName,
			Qty:         i.Qty,
		})
	}
	return &Request{
		ID:        id,
		Applicant: applicant,
		Items:     items,
		Status:    StatusNew,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Request) process(to Status, comment string) error {
	if r.Status.IsFinal() {
		return ErrAlreadyProcessed
	}
	now := time.Now()
	r.Status = to
	r.Comment = strings.TrimSpace(comment)
	r.ProcessedAt = &now
	return nil
}

func (r *Request) Approve(comment string) error {
	return r.process(StatusApproved, comment)
}

func (r *Request) Reject(comment string) error {
	return r.process(StatusRejected, comment)
}
