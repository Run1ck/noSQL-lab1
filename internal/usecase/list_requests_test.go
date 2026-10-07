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

func Test_ListRequests(t *testing.T) {
	userID := uuid.New()
	all := []*request.Request{newRequest(2, userID, request.StatusNew), newRequest(1, userID, request.StatusRejected)}

	postgres := new(mocks.Postgres)
	postgres.On("GetRequests", Any).Return(all, nil)
	postgres.On("GetRequestsByStatus", Any, request.StatusNew).Return(all[:1], nil)

	u := usecase.New(postgres, nil, nil, nil)
	ctx := context.Background()

	actual, err := u.ListRequests(ctx, dto.ListRequestsInput{})
	require.NoError(t, err)
	require.Len(t, actual.Requests, 2)

	actual, err = u.ListRequests(ctx, dto.ListRequestsInput{Status: "new"})
	require.NoError(t, err)
	require.Len(t, actual.Requests, 1)

	_, err = u.ListRequests(ctx, dto.ListRequestsInput{Status: "done"})
	require.ErrorIs(t, err, request.ErrInvalidStatus)
}
