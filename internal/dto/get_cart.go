package dto

import (
	"github.com/google/uuid"
)

type GetCartInput struct {
	UserID uuid.UUID
}

type CartItem struct {
	ServiceID string `json:"service_id"`
	Date      string `json:"date"`
}

type CartOutput struct {
	Items     []CartItem `json:"items"`
	ExpiresIn int        `json:"expires_in"`
}
