package postgres

import (
	"booking/internal/adapter/postgres/sqlc"
	"booking/internal/domain/user"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	q *sqlc.Queries
}

var _ user.Repository = (*UserRepo)(nil)

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{q: sqlc.New(pool)}
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
	if tag.RowsAffected() == 0 {
		return user.ErrLoginTaken
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomainUser(&row), nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	row, err := r.q.GetUserByLogin(ctx, login)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toDomainUser(&row), nil
}
