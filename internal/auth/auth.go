// Package auth описывает, кто делает запрос. Выпуск и проверку JWT
// (реализацию Tokens) пишет владелец авторизации; остальные модули берут
// пользователя только через FromContext.
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

// Principal — аутентифицированный пользователь запроса.
type Principal struct {
	UserID uuid.UUID
	Role   user.Role
}

func (p Principal) IsAdmin() bool {
	return p.Role == user.RoleAdmin
}

// Tokens выпускает и проверяет JWT: HS256 с JWT_SECRET, срок JWT_TTL,
// в claims sub = UserID, role = Role.
type Tokens interface {
	Issue(p Principal) (token string, expiresAt time.Time, err error)
	// Parse проверяет подпись и срок; любая проблема — ErrInvalidToken.
	Parse(token string) (Principal, error)
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext возвращает пользователя, которого положил middleware
// авторизации. За httpx.Middlewares.Auth ok всегда true.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
