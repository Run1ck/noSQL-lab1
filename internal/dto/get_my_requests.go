package dto

import (
	"github.com/google/uuid"
)

type GetMyRequestsInput struct {
	UserID uuid.UUID
}
