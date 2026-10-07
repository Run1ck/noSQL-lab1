package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) ApproveRequest(ctx context.Context, input dto.ProcessRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.postgres.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetRequest: %w", err)
	}

	err = req.Approve(input.AdminID, input.Comment)
	if err != nil {
		return output, fmt.Errorf("req.Approve: %w", err)
	}

	err = u.postgres.Approve(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.postgres.Approve: %w", err)
	}

	return toRequest(req), nil
}
