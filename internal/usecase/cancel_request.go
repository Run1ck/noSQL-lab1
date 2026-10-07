package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) CancelRequest(ctx context.Context, input dto.CancelRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.postgres.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetRequest: %w", err)
	}

	err = req.Cancel(input.UserID)
	if err != nil {
		return output, fmt.Errorf("req.Cancel: %w", err)
	}

	err = u.postgres.Cancel(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.postgres.Cancel: %w", err)
	}

	return toRequest(req), nil
}
