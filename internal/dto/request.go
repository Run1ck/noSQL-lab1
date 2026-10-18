package dto

import (
	"time"

	"github.com/google/uuid"
)

type RequestItem struct {
	ServiceID string `json:"service_id"`
	Date      string `json:"date"`
}

type RequestDetails struct {
	Purpose        string `json:"purpose"`
	Participants   int    `json:"participants"`
	NeedsProjector bool   `json:"needs_projector"`
}

type Request struct {
	ID          int64           `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Items       []RequestItem   `json:"items"`
	Details     *RequestDetails `json:"details"`
	Status      string          `json:"status"`
	Comment     string          `json:"comment"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at"`
	ProcessedBy *uuid.UUID      `json:"processed_by"`
}

type RequestsOutput struct {
	Requests []Request
}
