package dto

import (
	"github.com/google/uuid"
)

type GetMeInput struct {
	UserID uuid.UUID
}
