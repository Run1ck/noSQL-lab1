package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"booking/internal/domain/request"
	"booking/internal/dto"
	"booking/pkg/ratelimit"
)

func (u *UseCase) CreateRequest(ctx context.Context, input dto.CreateRequestInput) (dto.Request, error) {
	var output dto.Request

	details, err := newDetails(input.Details)
	if err != nil {
		return output, fmt.Errorf("request.NewDetails: %w", err)
	}

	c, err := u.redis.GetCart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCart: %w", err)
	}

	if c.IsEmpty() {
		return output, request.ErrEmptyCart
	}

	d, err := u.ban.Allow(ctx, input.UserID.String())
	if err != nil {
		slog.Warn("ban limit unavailable, request allowed", "err", err)
	} else if !d.Allowed {
		return output, &ratelimit.LimitedError{Banned: true, RetryAfter: d.RetryAfter}
	}

	req, err := request.NewFromCart(c, details)
	if err != nil {
		return output, fmt.Errorf("request.NewFromCart: %w", err)
	}

	err = u.postgres.CreateRequest(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.postgres.CreateRequest: %w", err)
	}

	err = u.redis.DeleteCart(ctx, input.UserID)
	if err != nil {
		slog.Warn("delete cart after request", "user_id", input.UserID, "err", err)
	}

	return toRequest(req), nil
}
