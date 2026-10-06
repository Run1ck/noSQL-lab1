// Package fake — репозитории в памяти для тестов use case'ов, HTTP-модулей
// и кэша: пишем против интерфейсов, не дожидаясь Postgres.
package fake

import (
	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"context"
	"slices"
	"sync"
)

// Services — service.Repository в памяти. Если задан Err, все методы
// возвращают его, как упавшая база.
type Services struct {
	mu    sync.Mutex
	items []*service.Service
	Err   error
}

func NewServices(items ...*service.Service) *Services {
	s := &Services{}
	for _, it := range items {
		_ = s.Save(context.Background(), it)
	}
	return s
}

func (s *Services) List(context.Context) ([]*service.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return nil, s.Err
	}
	out := make([]*service.Service, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, clone(it))
	}
	return out, nil
}

func (s *Services) Get(_ context.Context, id string) (*service.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return nil, s.Err
	}
	for _, it := range s.items {
		if it.ID == id {
			return clone(it), nil
		}
	}
	return nil, service.ErrNotFound
}

func (s *Services) Save(_ context.Context, svc *service.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return s.Err
	}
	for i, it := range s.items {
		if it.ID == svc.ID {
			s.items[i] = clone(svc)
			return nil
		}
	}
	s.items = append(s.items, clone(svc))
	return nil
}

// clone отдаёт копию: вызывающий не должен менять данные фейка в обход Save.
func clone(svc *service.Service) *service.Service {
	c := *svc
	return &c
}

// Bookings — booking.Repository в памяти. Свободный день отдаёт nil, как
// и разрешает контракт («пустой срез»), — вызывающие должны это пережить.
type Bookings struct {
	mu     sync.Mutex
	byDate map[service.Date][]booking.Booking
	Err    error
}

func NewBookings(items ...booking.Booking) *Bookings {
	b := &Bookings{byDate: make(map[service.Date][]booking.Booking)}
	for _, it := range items {
		b.byDate[it.Date] = append(b.byDate[it.Date], it)
	}
	return b
}

func (b *Bookings) ListByDate(_ context.Context, d service.Date) ([]booking.Booking, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.Err != nil {
		return nil, b.Err
	}
	return slices.Clone(b.byDate[d]), nil
}

// Date разбирает "YYYY-MM-DD" и паникует на ошибке — для тестовых данных.
func Date(s string) service.Date {
	d, err := service.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}
