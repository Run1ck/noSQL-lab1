package request

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, r *Request) error
	Get(ctx context.Context, id int64) (*Request, error)
	List(ctx context.Context) ([]*Request, error)
	ListByStatus(ctx context.Context, s Status) ([]*Request, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Request, error)

	Approve(ctx context.Context, r *Request) error
	Reject(ctx context.Context, r *Request) error
	Cancel(ctx context.Context, r *Request) error
}
