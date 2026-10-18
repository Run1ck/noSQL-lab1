package firestore

import (
	"errors"

	"cloud.google.com/go/firestore"

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
