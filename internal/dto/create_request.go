package dto

import (
	"github.com/google/uuid"
)

type CreateRequestInput struct {
	UserID  uuid.UUID       `json:"-"`
	Details *RequestDetails `json:"details"`
}
