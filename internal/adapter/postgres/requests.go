package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RequestRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

var _ request.Repository = (*RequestRepo)(nil)

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo {
	return &RequestRepo{pool: pool, q: sqlc.New(pool)}
}

func (r *RequestRepo) Create(ctx context.Context, req *request.Request) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	row, err := qtx.CreateRequest(ctx, req.UserID)
	if err != nil {
		return err
	}
	req.ID = row.ID
	req.CreatedAt = row.CreatedAt

	params := make([]sqlc.AddItemsToRequestParams, 0, len(req.Items))
	for _, item := range req.Items {
		params = append(params, sqlc.AddItemsToRequestParams{
			RequestID: req.ID,
			ServiceID: item.ServiceID,
			Date:      dateToDB(item.Date),
		})
	}
	if _, err := qtx.AddItemsToRequest(ctx, params); err != nil {
		if isPgCode(err, pgForeignKeyViolation) {
			return service.ErrNotFound
		}
		return err
	}
	return tx.Commit(ctx)
}

func (r *RequestRepo) Get(ctx context.Context, id int64) (*request.Request, error) {
	row, err := r.q.GetRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, request.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	itemRows, err := r.q.GetRequestItems(ctx, []int64{row.ID})
	if err != nil {
		return nil, err
	}
	return toDomainRequest(&row, itemRows), nil
}

func (r *RequestRepo) List(ctx context.Context) ([]*request.Request, error) {
	rows, err := r.q.GetAllRequests(ctx)
	if err != nil {
		return nil, err
	}
	return r.withItems(ctx, rows)
}

func (r *RequestRepo) ListByStatus(ctx context.Context, s request.Status) ([]*request.Request, error) {
	rows, err := r.q.GetRequestsByStatus(ctx, sqlc.RequestStatus(s))
	if err != nil {
		return nil, err
	}
	return r.withItems(ctx, rows)
}

func (r *RequestRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error) {
	rows, err := r.q.GetRequestsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return r.withItems(ctx, rows)
}

func (r *RequestRepo) Approve(ctx context.Context, req *request.Request) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	if err := process(ctx, qtx, req); err != nil {
		return err
	}
	if err := qtx.CreateBookingsForRequest(ctx, req.ID); err != nil {
		if isPgCode(err, pgUniqueViolation) {
			return request.ErrSlotBooked
		}
		return err
	}
	return tx.Commit(ctx)
}

func (r *RequestRepo) Reject(ctx context.Context, req *request.Request) error {
	return process(ctx, r.q, req)
}

func (r *RequestRepo) Cancel(ctx context.Context, req *request.Request) error {
	return process(ctx, r.q, req)
}

func process(ctx context.Context, q *sqlc.Queries, req *request.Request) error {
	tag, err := q.ProcessRequest(ctx, sqlc.ProcessRequestParams{
		ID:          req.ID,
		Status:      sqlc.RequestStatus(req.Status),
		Comment:     req.Comment,
		ProcessedAt: req.ProcessedAt,
		ProcessedBy: req.ProcessedBy,
	})
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return request.ErrAlreadyProcessed
	}
	return nil
}

func (r *RequestRepo) withItems(ctx context.Context, rows []sqlc.Request) ([]*request.Request, error) {
	if len(rows) == 0 {
		return []*request.Request{}, nil
	}

	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	itemRows, err := r.q.GetRequestItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make(map[int64][]sqlc.RequestItem, len(rows))
	for _, item := range itemRows {
		items[item.RequestID] = append(items[item.RequestID], item)
	}

	result := make([]*request.Request, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainRequest(&row, items[row.ID]))
	}
	return result, nil
}

func toDomainRequest(req *sqlc.Request, items []sqlc.RequestItem) *request.Request {
	return &request.Request{
		ID:          req.ID,
		UserID:      req.UserID,
		Items:       toDomainItems(items),
		Status:      request.Status(req.Status),
		Comment:     req.Comment,
		CreatedAt:   req.CreatedAt,
		ProcessedAt: req.ProcessedAt,
		ProcessedBy: req.ProcessedBy,
	}
}

func toDomainItems(items []sqlc.RequestItem) []request.Item {
	res := make([]request.Item, 0, len(items))
	for _, item := range items {
		res = append(res, request.Item{
			ServiceID: item.ServiceID,
			Date:      service.DateOf(item.Date),
		})
	}
	return res
}
