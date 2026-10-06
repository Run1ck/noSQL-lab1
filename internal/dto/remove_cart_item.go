package dto

import (
	"github.com/google/uuid"
)

type RemoveCartItemInput struct {
	UserID    uuid.UUID
	ServiceID string
	Date      string
}
