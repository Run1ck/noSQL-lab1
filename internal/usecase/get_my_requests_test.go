package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/request"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_GetMyRequests(t *testing.T) {
	userID := uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetRequestsByUser", Any, userID).Return([]*request.Request{
		newRequest(2, userID, request.StatusNew),
		newRequest(1, userID, request.StatusApproved),
	}, nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.GetMyRequests(context.Background(), dto.GetMyRequestsInput{UserID: userID})
	require.NoError(t, err)
	require.Len(t, actual.Requests, 2)
	require.Equal(t, int64(2), actual.Requests[0].ID)
	require.Equal(t, "approved", actual.Requests[1].Status)
}

func Test_GetMyRequests_Empty(t *testing.T) {
	userID := uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetRequestsByUser", Any, userID).Return([]*request.Request{}, nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.GetMyRequests(context.Background(), dto.GetMyRequestsInput{UserID: userID})
	require.NoError(t, err)
	require.NotNil(t, actual.Requests)
	require.Empty(t, actual.Requests)
}
