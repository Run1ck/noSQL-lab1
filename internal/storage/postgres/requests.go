package postgres

import (
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/storage/postgres/sqlc"
	"context"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RequestRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo {
	return &RequestRepo{pool, sqlc.New(pool)}
}

func (r *RequestRepo) Create(ctx context.Context, req *request.Request) error {
	_, err := r.q.CreateRequest(ctx, req.UserID)
	return err
}

func (r *RequestRepo) Get(ctx context.Context, id int64) (*request.Request, error) {
	row, err := r.q.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	items_row, err := r.q.GetRequestItems(ctx, []int64{row.ID})
	return toDomainRequest(&row, items_row), err
}

func toDomainItems(items []sqlc.RequestItem) []request.Item {
	res := make([]request.Item, 0, len(items))
	for _, item := range items {
		res = append(res, request.Item{ServiceID: item.ServiceID,
			Date: service.DateOf(item.Date)})
	}
	return res
}

func toDomainRequest(req *sqlc.Request, items []sqlc.RequestItem) *request.Request {
	return &request.Request{ID: req.ID,
		UserID:      req.UserID,
		Items:       toDomainItems(items),
		Status:      request.Status(req.Status),
		Comment:     req.Comment,
		CreatedAt:   req.CreatedAt,
		ProcessedAt: req.ProcessedAt,
		ProcessedBy: req.ProcessedBy,
	}
}

func (r *RequestRepo) GetRequestsDomain(ctx context.Context, row []sqlc.Request) ([]*request.Request, error) {

	ids := make([]int64, 0, len(row))
	for i := 0; i < len(row); i++ {
		ids = append(ids, row[i].ID)
	}
	items_row, err := r.q.GetRequestItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make(map[int64][]sqlc.RequestItem)
	for _, item := range items_row {
		items[item.RequestID] = append(items[item.RequestID], item)
	}

	result := make([]*request.Request, 0, len(row))
	for _, req := range row {
		result = append(result, toDomainRequest(&req, items[req.ID]))
	}
	return result, nil
}

func (r *RequestRepo) List(ctx context.Context) ([]*request.Request, error) {
	row, err := r.q.GetAllRequests(ctx)
	if err != nil {
		return nil, err
	}

	return r.GetRequestsDomain(ctx, row)
}

func (r *RequestRepo) ListByStatus(ctx context.Context, s request.Status) ([]*request.Request, error) {
	var status sqlc.RequestStatus
	status.Scan(s)
	row, err := r.q.GetRequestsByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	return r.GetRequestsDomain(ctx, row)
}

func (r *RequestRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error) {
	row, err := r.q.GetRequestsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return r.GetRequestsDomain(ctx, row)
}

func (r *RequestRepo) ProcessRequest(ctx context.Context, req *request.Request) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	r.q.WithTx(tx).ProcessRequest(ctx, sqlc.ProcessRequestParams{
		ID:          req.ID,
		Status:      sqlc.RequestStatus(req.Status),
		Comment:     req.Comment,
		ProcessedAt: req.ProcessedAt,
		ProcessedBy: req.ProcessedBy,
	})
	r.q.WithTx(tx).CreateBookingsForRequest(ctx, req.ID)

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *RequestRepo) Approve(ctx context.Context, req *request.Request) error {
	req.Status = request.StatusApproved
	return r.ProcessRequest(ctx, req)
}

func (r *RequestRepo) Reject(ctx context.Context, req *request.Request) error {
	req.Status = request.StatusRejected
	return r.ProcessRequest(ctx, req)
}

func (r *RequestRepo) Cancel(ctx context.Context, req *request.Request) error {
	req.Status = request.StatusCancelled
	return r.ProcessRequest(ctx, req)
}
