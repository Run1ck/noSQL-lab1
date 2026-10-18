package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) CancelRequest(ctx context.Context, input dto.CancelRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.requests.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.requests.GetRequest: %w", err)
	}

	err = req.Cancel(input.UserID)
	if err != nil {
		return output, fmt.Errorf("req.Cancel: %w", err)
	}

	err = u.requests.Cancel(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.requests.Cancel: %w", err)
	}

	return toRequest(req), nil
}
