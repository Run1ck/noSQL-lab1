package dto

import (
	"github.com/google/uuid"
)

type User struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
	Role  string    `json:"role"`
}
