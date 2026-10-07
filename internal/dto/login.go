package dto

import (
	"time"
)

type LoginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}
