package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/service"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

const Any = mock.Anything

func Test_GetServices_Success(t *testing.T) {
	// Данные для поведения
	services := []*service.Service{
		{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
		{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
	}

	// Настраиваем поведение Postgres
	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return(services, nil)
	defer postgres.AssertCalled(t, "GetServices", Any)

	// Собираем UseCase
	u := usecase.New(postgres, nil, nil, nil)

	{ // Сам тест: только активные
		output := dto.GetServicesOutput{Services: []dto.Service{
			{ID: "room-101", Name: "Аудитория 101", Kind: "room", Active: true},
		}}

		actual, err := u.GetServices(context.Background())
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_GetServices_Empty(t *testing.T) {
	// Настраиваем поведение Postgres
	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return([]*service.Service{}, nil)

	// Собираем UseCase
	u := usecase.New(postgres, nil, nil, nil)

	{ // Сам тест: пустой срез, а не nil — в JSON будет [], а не null
		actual, err := u.GetServices(context.Background())
		require.NoError(t, err)
		require.Equal(t, dto.GetServicesOutput{Services: []dto.Service{}}, actual)
	}
}

func Test_GetServices_Error(t *testing.T) {
	// Данные для поведения
	errDB := errors.New("db down")

	// Настраиваем поведение Postgres
	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return(nil, errDB)

	// Собираем UseCase
	u := usecase.New(postgres, nil, nil, nil)

	{ // Сам тест
		_, err := u.GetServices(context.Background())
		require.ErrorIs(t, err, errDB)
	}
}
