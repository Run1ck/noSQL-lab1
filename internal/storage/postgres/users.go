package postgres

import (
	"booking/internal/domain/user"
	"booking/internal/storage/postgres/sqlc"
	"context"
	"errors"

	"github.com/google/uuid"
)

type UserRepo struct {
	q *sqlc.Queries
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	tag, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
		Role:         sqlc.UserRole(u.Role),
		CreatedAt:    u.CreatedAt,
	})

	if err != nil {
		return err
	}

	if tag.RowsAffected() != 1 {
		return errors.New("Failed to add a new user with these credentials")
	}

	return nil
}

func toDomainUser(u *sqlc.User) *user.User {
	return &user.User{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
		Role:         user.Role(u.Role),
		CreatedAt:    u.CreatedAt,
	}
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toDomainUser(&row), nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	row, err := r.q.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, err
	}

	return toDomainUser(&row), nil
}
