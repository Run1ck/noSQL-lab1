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
	services := []*service.Service{
		{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
		{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
	}

	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return(services, nil)
	defer postgres.AssertCalled(t, "GetServices", Any)

	u := usecase.New(postgres, nil, nil, nil)

	{
		output := dto.GetServicesOutput{Services: []dto.Service{
			{ID: "room-101", Name: "Аудитория 101", Kind: "room", Active: true},
		}}

		actual, err := u.GetServices(context.Background())
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_GetServices_Empty(t *testing.T) {
	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return([]*service.Service{}, nil)

	u := usecase.New(postgres, nil, nil, nil)

	{
		actual, err := u.GetServices(context.Background())
		require.NoError(t, err)
		require.Equal(t, dto.GetServicesOutput{Services: []dto.Service{}}, actual)
	}
}

func Test_GetServices_Error(t *testing.T) {
	errDB := errors.New("db down")

	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return(nil, errDB)

	u := usecase.New(postgres, nil, nil, nil)

	{
		_, err := u.GetServices(context.Background())
		require.ErrorIs(t, err, errDB)
	}
}
