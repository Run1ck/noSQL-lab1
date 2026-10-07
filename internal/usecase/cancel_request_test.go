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

func Test_CancelRequest_Success(t *testing.T) {
	owner := uuid.New()

	postgres := new(mocks.Postgres)
	postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, owner, request.StatusNew), nil)
	postgres.On("Cancel", Any, mock.MatchedBy(func(r *request.Request) bool {
		return r.Status == request.StatusCancelled && *r.ProcessedBy == owner && r.ProcessedAt != nil
	})).Return(nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.CancelRequest(context.Background(), dto.CancelRequestInput{UserID: owner, ID: 42})
	require.NoError(t, err)
	require.Equal(t, "cancelled", actual.Status)
	require.Equal(t, owner, *actual.ProcessedBy)
}

func Test_CancelRequest_Rejected(t *testing.T) {
	owner := uuid.New()

	for _, tc := range []struct {
		name    string
		userID  uuid.UUID
		status  request.Status
		saveErr error
		err     error
	}{
		{name: "not owner", userID: uuid.New(), status: request.StatusNew, err: request.ErrNotOwner},
		{name: "already processed", userID: owner, status: request.StatusApproved, err: request.ErrAlreadyProcessed},
		{name: "processed concurrently", userID: owner, status: request.StatusNew, saveErr: request.ErrAlreadyProcessed, err: request.ErrAlreadyProcessed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			postgres := new(mocks.Postgres)
			postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, owner, tc.status), nil)
			postgres.On("Cancel", Any, Any).Return(tc.saveErr)

			u := usecase.New(postgres, nil, nil, nil)

			_, err := u.CancelRequest(context.Background(), dto.CancelRequestInput{UserID: tc.userID, ID: 42})
			require.ErrorIs(t, err, tc.err)
			if tc.saveErr == nil {
				postgres.AssertNotCalled(t, "Cancel", Any, Any)
			}
		})
	}
}
