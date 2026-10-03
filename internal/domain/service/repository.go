package service

import "context"

type Repository interface {
	// List возвращает все услуги, включая неактивные; каталог фильтрует сам.
	List(ctx context.Context) ([]*Service, error)
	// Get возвращает ErrNotFound, если услуги нет.
	Get(ctx context.Context, id string) (*Service, error)
	// Save создаёт услугу или обновляет существующую с тем же ID.
	Save(ctx context.Context, service *Service) error
}
