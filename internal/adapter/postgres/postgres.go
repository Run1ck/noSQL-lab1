package postgres

import (
	"booking/internal/domain/booking"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/internal/usecase"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	services *ServiceRepo
	bookings *BookingRepo
	requests *RequestRepo
	users    *UserRepo
}

var _ usecase.Postgres = (*Postgres)(nil)

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{
		services: NewServiceRepo(pool),
		bookings: NewBookingRepo(pool),
		requests: NewRequestRepo(pool),
		users:    NewUserRepo(pool),
	}
}

func (p *Postgres) GetServices(ctx context.Context) ([]*service.Service, error) {
	return p.services.List(ctx)
}

func (p *Postgres) GetService(ctx context.Context, id string) (*service.Service, error) {
	return p.services.Get(ctx, id)
}

func (p *Postgres) SaveService(ctx context.Context, s *service.Service) error {
	return p.services.Save(ctx, s)
}

func (p *Postgres) GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error) {
	return p.bookings.ListByDate(ctx, date)
}

func (p *Postgres) Approve(ctx context.Context, req *request.Request) error {
	return p.requests.Approve(ctx, req)
}

func (p *Postgres) Reject(ctx context.Context, req *request.Request) error {
	return p.requests.Reject(ctx, req)
}

func (p *Postgres) Cancel(ctx context.Context, req *request.Request) error {
	return p.requests.Cancel(ctx, req)
}

func (p *Postgres) CreateRequest(ctx context.Context, req *request.Request) error {
	return p.requests.Create(ctx, req)
}

func (p *Postgres) GetRequest(ctx context.Context, id int64) (*request.Request, error) {
	return p.requests.Get(ctx, id)
}

func (p *Postgres) GetRequests(ctx context.Context) ([]*request.Request, error) {
	return p.requests.List(ctx)
}

func (p *Postgres) GetRequestsByStatus(ctx context.Context, s request.Status) ([]*request.Request, error) {
	return p.requests.ListByStatus(ctx, s)
}

func (p *Postgres) GetRequestsByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error) {
	return p.requests.ListByUser(ctx, userID)
}

func (p *Postgres) GetUserByLogin(ctx context.Context, login string) (*user.User, error) {
	return p.users.GetByLogin(ctx, login)
}

func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return p.users.GetByID(ctx, id)
}

func (p *Postgres) CreateUser(ctx context.Context, u *user.User) error {
	return p.users.Create(ctx, u)
}
