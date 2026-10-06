package dto

import (
	"github.com/google/uuid"
)

type AddCartItemInput struct {
	UserID    uuid.UUID `json:"-"`
	ServiceID string    `json:"service_id"`
	Date      string    `json:"date"`
}
