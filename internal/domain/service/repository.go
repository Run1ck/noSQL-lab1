package service

import "context"

type Repository interface {
	List(ctx context.Context) ([]*Service, error)
	Get(ctx context.Context, id string) (*Service, error)
	Save(ctx context.Context, service *Service) error
}
