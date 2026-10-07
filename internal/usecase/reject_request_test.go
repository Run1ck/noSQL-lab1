package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/request"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func Test_RejectRequest(t *testing.T) {
	adminID := uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, uuid.New(), request.StatusNew), nil).Once()
	postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, uuid.New(), request.StatusApproved), nil).Once()
	postgres.On("Reject", Any, mock.MatchedBy(func(r *request.Request) bool {
		return r.Status == request.StatusRejected && *r.ProcessedBy == adminID
	})).Return(nil).Once()

	u := usecase.New(postgres, nil, nil, nil)
	ctx := context.Background()

	actual, err := u.RejectRequest(ctx, dto.ProcessRequestInput{AdminID: adminID, ID: 42, Comment: "busy"})
	require.NoError(t, err)
	require.Equal(t, "rejected", actual.Status)
	require.Equal(t, "busy", actual.Comment)

	_, err = u.RejectRequest(ctx, dto.ProcessRequestInput{AdminID: adminID, ID: 42})
	require.ErrorIs(t, err, request.ErrAlreadyProcessed)
	postgres.AssertNumberOfCalls(t, "Reject", 1)
}
