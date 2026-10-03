package user

import "errors"

var (
	ErrInvalidPassword    = errors.New("password must be 8 to 72 bytes long")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotFound           = errors.New("user not found")
	ErrLoginTaken         = errors.New("login is already taken")
)
