package user

import "fmt"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleUser, RoleAdmin:
		return Role(s), nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
}
