package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// Create сохраняет нового пользователя; если логин уже занят — ErrLoginTaken.
	Create(ctx context.Context, u *User) error
	// GetByID и GetByLogin возвращают ErrNotFound, если пользователя нет.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
}
