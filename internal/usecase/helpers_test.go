package usecase_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	"booking/internal/auth"
	"booking/internal/domain/request"
	"booking/pkg/ratelimit"
)

const testJWTSecret = "test-secret-at-least-32-bytes-long!"

func newTokens() auth.Tokens {
	return auth.NewJWT(testJWTSecret, time.Hour)
}

type fakeBan struct {
	d     ratelimit.Decision
	err   error
	calls int
}

func (f *fakeBan) Allow(_ context.Context, _ string) (ratelimit.Decision, error) {
	f.calls++
	return f.d, f.err
}

func newRequest(id int64, userID uuid.UUID, status request.Status, items ...request.Item) *request.Request {
	return &request.Request{
		ID:        id,
		UserID:    userID,
		Items:     items,
		Status:    status,
		CreatedAt: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC),
	}
}
