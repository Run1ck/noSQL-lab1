package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"booking/internal/domain/service"
	"booking/internal/usecase"
)

const servicesKey = "cache:services"

type Catalog struct {
	usecase.Catalog
	cache
}

var _ usecase.Catalog = (*Catalog)(nil)

func NewCatalog(client *redis.Client, catalog usecase.Catalog, ttl time.Duration) *Catalog {
	return &Catalog{
		Catalog: catalog,
		cache:   cache{redis: client, ttl: ttl},
	}
}

func (c *Catalog) GetServices(ctx context.Context) ([]*service.Service, error) {
	var services []*service.Service
	if c.load(ctx, servicesKey, &services) {
		return services, nil
	}

	services, err := c.Catalog.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	c.store(ctx, servicesKey, services)

	return services, nil
}

func (c *Catalog) GetService(ctx context.Context, id string) (*service.Service, error) {
	services, err := c.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range services {
		if s.ID == id {
			return s, nil
		}
	}

	return c.Catalog.GetService(ctx, id)
}

func (c *Catalog) SaveService(ctx context.Context, s *service.Service) error {
	err := c.Catalog.SaveService(ctx, s)
	if err != nil {
		return err
	}

	c.invalidate(ctx, servicesKey)

	return nil
}
