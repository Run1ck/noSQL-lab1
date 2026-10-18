package firestore

import (
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"booking/internal/usecase"
)

var errNotImplemented = errors.New("firestore: not implemented yet")

type Firestore struct {
	client *firestore.Client
}

var _ usecase.Requests = (*Firestore)(nil)

func New(client *firestore.Client) *Firestore {
	return &Firestore{client: client}
}

func isCode(err error, code codes.Code) bool {
	return status.Code(err) == code
}
