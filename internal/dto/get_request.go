package dto

import (
	"github.com/google/uuid"
)

type GetRequestInput struct {
	UserID  uuid.UUID
	IsAdmin bool
	ID      int64
}
