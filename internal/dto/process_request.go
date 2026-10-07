package dto

import (
	"github.com/google/uuid"
)

type ProcessRequestInput struct {
	AdminID uuid.UUID `json:"-"`
	ID      int64     `json:"-"`
	Comment string    `json:"comment"`
}
