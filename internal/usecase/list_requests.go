package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/request"
	"booking/internal/dto"
)

func (u *UseCase) ListRequests(ctx context.Context, input dto.ListRequestsInput) (dto.RequestsOutput, error) {
	if input.Status == "" {
		reqs, err := u.postgres.GetRequests(ctx)
		if err != nil {
			return dto.RequestsOutput{}, fmt.Errorf("u.postgres.GetRequests: %w", err)
		}

		return toRequests(reqs), nil
	}

	status, err := request.ParseStatus(input.Status)
	if err != nil {
		return dto.RequestsOutput{}, fmt.Errorf("request.ParseStatus: %w", err)
	}

	reqs, err := u.postgres.GetRequestsByStatus(ctx, status)
	if err != nil {
		return dto.RequestsOutput{}, fmt.Errorf("u.postgres.GetRequestsByStatus: %w", err)
	}

	return toRequests(reqs), nil
}
