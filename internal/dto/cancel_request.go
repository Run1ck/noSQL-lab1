package dto

import (
	"github.com/google/uuid"
)

type CancelRequestInput struct {
	UserID uuid.UUID
	ID     int64
}
