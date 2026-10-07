package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"booking/internal/domain/service"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_GetAllServices_IncludesInactive(t *testing.T) {
	postgres := new(mocks.Postgres)
	postgres.On("GetServices", Any).Return([]*service.Service{
		{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
		{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
	}, nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.GetAllServices(context.Background())
	require.NoError(t, err)
	require.Equal(t, []dto.Service{
		{ID: "lab-1", Name: "Лаборатория", Kind: "lab", Active: false},
		{ID: "room-101", Name: "Аудитория 101", Kind: "room", Active: true},
	}, actual.Services)
}
