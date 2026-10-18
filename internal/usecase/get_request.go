package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/request"
	"booking/internal/dto"
)

func (u *UseCase) GetRequest(ctx context.Context, input dto.GetRequestInput) (dto.Request, error) {
	var output dto.Request

	req, err := u.requests.GetRequest(ctx, input.ID)
	if err != nil {
		return output, fmt.Errorf("u.requests.GetRequest: %w", err)
	}

	if !input.IsAdmin && req.UserID != input.UserID {
		return output, request.ErrNotOwner
	}

	return toRequest(req), nil
}
