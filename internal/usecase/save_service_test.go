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

func Test_SaveService_Success(t *testing.T) {
	postgres := new(mocks.Postgres)
	postgres.On("SaveService", Any, &service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true}).Return(nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.SaveService(context.Background(), dto.SaveServiceInput{ID: "room-101", Name: " Аудитория 101 ", Kind: "room", Active: true})
	require.NoError(t, err)
	require.Equal(t, dto.Service{ID: "room-101", Name: "Аудитория 101", Kind: "room", Active: true}, actual)
}

func Test_SaveService_Rejected(t *testing.T) {
	postgres := new(mocks.Postgres)
	u := usecase.New(postgres, nil, nil, nil)
	ctx := context.Background()

	_, err := u.SaveService(ctx, dto.SaveServiceInput{ID: "room-101", Name: "Аудитория", Kind: "garage"})
	require.ErrorIs(t, err, service.ErrInvalidKind)

	_, err = u.SaveService(ctx, dto.SaveServiceInput{ID: "room-101", Name: " ", Kind: "room"})
	require.ErrorIs(t, err, service.ErrInvalid)

	postgres.AssertNotCalled(t, "SaveService", Any, Any)
}
