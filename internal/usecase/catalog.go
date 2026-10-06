package usecase

import (
	"booking/internal/domain/service"
	"context"
)

// Catalog — каталог услуг для пользователей.
type Catalog struct {
	services service.Repository
}

// NewCatalog принимает service.Repository; в main это кэш-декоратор поверх Postgres.
func NewCatalog(services service.Repository) *Catalog {
	return &Catalog{services: services}
}

// Active — услуги, которые можно бронировать, в порядке List. Неактивные
// видит только админка.
func (uc *Catalog) Active(ctx context.Context) ([]*service.Service, error) {
	all, err := uc.services.List(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]*service.Service, 0, len(all))
	for _, s := range all {
		if s.Active {
			active = append(active, s)
		}
	}
	return active, nil
}
