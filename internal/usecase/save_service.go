package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) SaveService(ctx context.Context, input dto.SaveServiceInput) (dto.Service, error) {
	var output dto.Service

	kind, err := service.ParseKind(input.Kind)
	if err != nil {
		return output, fmt.Errorf("service.ParseKind: %w", err)
	}

	s, err := service.New(input.ID, input.Name, kind, input.Active)
	if err != nil {
		return output, fmt.Errorf("service.New: %w", err)
	}

	err = u.catalog.SaveService(ctx, s)
	if err != nil {
		return output, fmt.Errorf("u.catalog.SaveService: %w", err)
	}

	return toService(s), nil
}
