package user

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

func New(login, password string, role Role) (*User, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if _, err := ParseRole(string(role)); err != nil {
		return nil, err
	}
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return nil, ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now(),
	}, nil
}

func (u *User) CheckPassword(password string) error {
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
