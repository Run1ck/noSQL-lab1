package auth

import (
	"booking/internal/domain/user"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrForbidden    = errors.New("admin role required")
)

type Principal struct {
	UserID uuid.UUID
	Role   user.Role
}

func (p Principal) IsAdmin() bool {
	return p.Role == user.RoleAdmin
}

type Tokens interface {
	Issue(p Principal) (token string, expiresAt time.Time, err error)
	Parse(token string) (Principal, error)
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
