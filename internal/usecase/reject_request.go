package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) RejectRequest(ctx context.Context, input dto.ProcessRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.postgres.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetRequest: %w", err)
	}

	err = req.Reject(input.AdminID, input.Comment)
	if err != nil {
		return output, fmt.Errorf("req.Reject: %w", err)
	}

	err = u.postgres.Reject(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.postgres.Reject: %w", err)
	}

	return toRequest(req), nil
}
