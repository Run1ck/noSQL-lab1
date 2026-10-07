package usecase

import (
	"context"
	"fmt"

	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) GetAllServices(ctx context.Context) (dto.GetServicesOutput, error) {
	var output dto.GetServicesOutput

	services, err := u.postgres.GetServices(ctx)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetServices: %w", err)
	}

	output.Services = make([]dto.Service, 0, len(services))
	for _, s := range services {
		output.Services = append(output.Services, toService(s))
	}

	return output, nil
}

func toService(s *service.Service) dto.Service {
	return dto.Service{ID: s.ID, Name: s.Name, Kind: string(s.Kind), Active: s.Active}
}
