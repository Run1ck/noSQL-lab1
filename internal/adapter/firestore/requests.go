package firestore

import (
	"context"

	"github.com/google/uuid"

	"booking/internal/domain/request"
)

func (f *Firestore) CreateRequest(ctx context.Context, req *request.Request) error {
	return errNotImplemented
}

func (f *Firestore) GetRequest(ctx context.Context, id int64) (*request.Request, error) {
	return nil, errNotImplemented
}

func (f *Firestore) GetRequests(ctx context.Context) ([]*request.Request, error) {
	return nil, errNotImplemented
}

func (f *Firestore) GetRequestsByStatus(ctx context.Context, s request.Status) ([]*request.Request, error) {
	return nil, errNotImplemented
}

func (f *Firestore) GetRequestsByUser(ctx context.Context, userID uuid.UUID) ([]*request.Request, error) {
	return nil, errNotImplemented
}

func (f *Firestore) Approve(ctx context.Context, req *request.Request) error {
	return errNotImplemented
}

func (f *Firestore) Reject(ctx context.Context, req *request.Request) error {
	return errNotImplemented
}

func (f *Firestore) Cancel(ctx context.Context, req *request.Request) error {
	return errNotImplemented
}
