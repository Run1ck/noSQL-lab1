package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) RejectRequest(ctx context.Context, input dto.ProcessRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.requests.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.requests.GetRequest: %w", err)
	}

	err = req.Reject(input.AdminID, input.Comment)
	if err != nil {
		return output, fmt.Errorf("req.Reject: %w", err)
	}

	err = u.requests.Reject(ctx, req)
	if err != nil {
		return output, fmt.Errorf("u.requests.Reject: %w", err)
	}

	return toRequest(req), nil
}
