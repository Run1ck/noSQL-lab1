package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"booking/internal/auth"
	"booking/internal/domain/booking"
	"booking/internal/domain/cart"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/pkg/ratelimit"
)

type Redis interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error)
	SaveCart(ctx context.Context, c *cart.Cart) error
	RemoveCartItem(ctx context.Context, userID uuid.UUID, item cart.Item) error
	DeleteCart(ctx context.Context, userID uuid.UUID) error
	GetCartTTL(ctx context.Context, userID uuid.UUID) (time.Duration, error)
}

type Users interface {
	GetUserByLogin(ctx context.Context, login string) (*user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	CreateUser(ctx context.Context, u *user.User) error
}

type Requests interface {
	CreateRequest(ctx context.Context, req *request.Request) error
	GetRequest(ctx context.Context, id int64) (*request.Request, error)
	GetRequests(ctx context.Context) ([]*request.Request, error)
	GetRequestsByStatus(ctx context.Context, s request.Status) ([]*request.Request, error)
	GetRequestsByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error)

	Approve(ctx context.Context, req *request.Request) error
	Reject(ctx context.Context, req *request.Request) error
	Cancel(ctx context.Context, req *request.Request) error

	GetBookings(ctx context.Context, date service.Date) ([]booking.Booking, error)
}

type Catalog interface {
	GetServices(ctx context.Context) ([]*service.Service, error)
	GetService(ctx context.Context, id string) (*service.Service, error)
	SaveService(ctx context.Context, s *service.Service) error
}

type UseCase struct {
	users    Users
	requests Requests
	catalog  Catalog
	redis    Redis
	tokens   auth.Tokens
	ban      ratelimit.Limiter
}

func New(users Users, requests Requests, catalog Catalog, redis Redis, tokens auth.Tokens, ban ratelimit.Limiter) *UseCase {
	return &UseCase{
		users:    users,
		requests: requests,
		catalog:  catalog,
		redis:    redis,
		tokens:   tokens,
		ban:      ban,
	}
}
