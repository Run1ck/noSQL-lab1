package request

import "context"

type Repository interface {
	NextID(ctx context.Context) (int64, error)
	Save(ctx context.Context, r *Request) error
	Get(ctx context.Context, id int64) (*Request, error)
	List(ctx context.Context) ([]*Request, error)
	ListByStatus(ctx context.Context, s Status) ([]*Request, error)
}
