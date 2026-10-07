package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/service"
	"context"
)

type ServiceRepo struct {
	q *sqlc.Queries
}

func toDomainService(s *sqlc.Service) *service.Service {
	return &service.Service{ID: s.ID,
		Name:   s.Name,
		Kind:   service.Kind(s.Kind),
		Active: s.Active}
}

func (r *ServiceRepo) List(ctx context.Context) ([]*service.Service, error) {
	row, err := r.q.GetAllServices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*service.Service, 0, len(row))
	for _, s := range row {
		result = append(result, toDomainService(&s))
	}

	return result, nil
}

func (r *ServiceRepo) Get(ctx context.Context, id string) (*service.Service, error) {
	row, err := r.q.GetService(ctx, id)
	if err != nil {
		return nil, err
	}

	return toDomainService(&row), nil
}

func (r *ServiceRepo) Save(ctx context.Context, s *service.Service) error {
	err := r.q.UpsertService(ctx, sqlc.UpsertServiceParams{
		ID:     s.ID,
		Name:   s.Name,
		Kind:   sqlc.ServiceKind(s.Kind),
		Active: s.Active,
	})
	if err != nil {
		return err
	}

	return nil
}
