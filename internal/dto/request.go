package dto

import (
	"time"

	"github.com/google/uuid"
)

type RequestItem struct {
	ServiceID string `json:"service_id"`
	Date      string `json:"date"`
}

type Request struct {
	ID          int64         `json:"id"`
	UserID      uuid.UUID     `json:"user_id"`
	Items       []RequestItem `json:"items"`
	Status      string        `json:"status"`
	Comment     string        `json:"comment"`
	CreatedAt   time.Time     `json:"created_at"`
	ProcessedAt *time.Time    `json:"processed_at"`
	ProcessedBy *uuid.UUID    `json:"processed_by"`
}

type RequestsOutput struct {
	Requests []Request
}
