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

func Test_ApproveRequest_Success(t *testing.T) {
	adminID := uuid.New()
	item := request.Item{ServiceID: "room-101", Date: date("2026-10-05")}

	postgres := new(mocks.Postgres)
	postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, uuid.New(), request.StatusNew, item), nil)
	postgres.On("Approve", Any, mock.MatchedBy(func(r *request.Request) bool {
		return r.Status == request.StatusApproved && *r.ProcessedBy == adminID && r.Comment == "ok"
	})).Return(nil)

	u := usecase.New(postgres, nil, nil, nil)

	actual, err := u.ApproveRequest(context.Background(), dto.ProcessRequestInput{AdminID: adminID, ID: 42, Comment: " ok "})
	require.NoError(t, err)
	require.Equal(t, "approved", actual.Status)
	require.Equal(t, "ok", actual.Comment)
	require.Equal(t, []dto.RequestItem{{ServiceID: "room-101", Date: "2026-10-05"}}, actual.Items)
}

func Test_ApproveRequest_Rejected(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  request.Status
		getErr  error
		saveErr error
		err     error
	}{
		{name: "not found", getErr: request.ErrNotFound, err: request.ErrNotFound},
		{name: "already processed", status: request.StatusRejected, err: request.ErrAlreadyProcessed},
		{name: "slot booked", status: request.StatusNew, saveErr: request.ErrSlotBooked, err: request.ErrSlotBooked},
		{name: "processed concurrently", status: request.StatusNew, saveErr: request.ErrAlreadyProcessed, err: request.ErrAlreadyProcessed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			postgres := new(mocks.Postgres)
			if tc.getErr != nil {
				postgres.On("GetRequest", Any, int64(42)).Return(nil, tc.getErr)
			} else {
				postgres.On("GetRequest", Any, int64(42)).Return(newRequest(42, uuid.New(), tc.status), nil)
			}
			postgres.On("Approve", Any, Any).Return(tc.saveErr)

			u := usecase.New(postgres, nil, nil, nil)

			_, err := u.ApproveRequest(context.Background(), dto.ProcessRequestInput{AdminID: uuid.New(), ID: 42})
			require.ErrorIs(t, err, tc.err)
		})
	}
}
