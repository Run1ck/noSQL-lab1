package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

func (u *UseCase) GetMyRequests(ctx context.Context, input dto.GetMyRequestsInput) (dto.RequestsOutput, error) {
	reqs, err := u.postgres.GetRequestsByUser(ctx, input.UserID)
	if err != nil {
		return dto.RequestsOutput{}, fmt.Errorf("u.postgres.GetRequestsByUser: %w", err)
	}

	return toRequests(reqs), nil
}
