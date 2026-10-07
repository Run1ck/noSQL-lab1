package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/service"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceRepo struct {
	q *sqlc.Queries
}

var _ service.Repository = (*ServiceRepo)(nil)

func NewServiceRepo(pool *pgxpool.Pool) *ServiceRepo {
	return &ServiceRepo{q: sqlc.New(pool)}
}

func toDomainService(s *sqlc.Service) *service.Service {
	return &service.Service{
		ID:     s.ID,
		Name:   s.Name,
		Kind:   service.Kind(s.Kind),
		Active: s.Active,
	}
}

func (r *ServiceRepo) List(ctx context.Context) ([]*service.Service, error) {
	rows, err := r.q.GetAllServices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*service.Service, 0, len(rows))
	for _, s := range rows {
		result = append(result, toDomainService(&s))
	}
	return result, nil
}

func (r *ServiceRepo) Get(ctx context.Context, id string) (*service.Service, error) {
	row, err := r.q.GetService(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomainService(&row), nil
}

func (r *ServiceRepo) Save(ctx context.Context, s *service.Service) error {
	return r.q.UpsertService(ctx, sqlc.UpsertServiceParams{
		ID:     s.ID,
		Name:   s.Name,
		Kind:   sqlc.ServiceKind(s.Kind),
		Active: s.Active,
	})
}
