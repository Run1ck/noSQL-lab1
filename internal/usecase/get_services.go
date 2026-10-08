package usecase

import (
	"context"
	"fmt"

	"booking/internal/dto"
)

// GetServices — каталог: только активные услуги, неактивные видит админка.
func (u *UseCase) GetServices(ctx context.Context) (dto.GetServicesOutput, error) {
	var output dto.GetServicesOutput

	services, err := u.postgres.GetServices(ctx)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetServices: %w", err)
	}

	output.Services = make([]dto.Service, 0, len(services))

	for _, s := range services {
		if !s.Active {
			continue
		}

		output.Services = append(output.Services, toService(s))
	}

	return output, nil
}
