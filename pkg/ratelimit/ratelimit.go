package ratelimit

import (
	"context"
	"fmt"
	"time"
)

type Decision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(ctx context.Context, key string) (Decision, error)
}

type LimitedError struct {
	Banned     bool
	RetryAfter time.Duration
}

func (e *LimitedError) Error() string {
	if e.Banned {
		return fmt.Sprintf("too many requests, banned for %s", e.RetryAfter.Round(time.Second))
	}
	return fmt.Sprintf("rate limit exceeded, retry after %s", e.RetryAfter.Round(time.Second))
}
