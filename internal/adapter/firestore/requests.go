package firestore

import (
	"context"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"booking/internal/domain/request"
)

const (
	requestsCollection = "requests"
	countersCollection = "counters"
	requestsCounterID  = "requests"
)

type counterDoc struct {
	Last int64 `firestore:"last"`
}

func (f *Firestore) requestRef(id int64) *firestore.DocumentRef {
	return f.client.Collection(requestsCollection).Doc(strconv.FormatInt(id, 10))
}

func (f *Firestore) CreateRequest(ctx context.Context, req *request.Request) error {
	req.CreatedAt = req.CreatedAt.UTC().Truncate(time.Microsecond)
	counterRef := f.client.Collection(countersCollection).Doc(requestsCounterID)

	var id int64
	err := f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var counter counterDoc
		snap, err := tx.Get(counterRef)
		switch {
		case isCode(err, codes.NotFound):
			// счётчика ещё нет: первая заявка, начинаем с нуля
		case err != nil:
			return err
		default:
			err = snap.DataTo(&counter)
			if err != nil {
				return err
			}
		}

		id = counter.Last + 1

		err = tx.Set(counterRef, counterDoc{Last: id})
		if err != nil {
			return err
		}

		return tx.Create(f.requestRef(id), newRequestDoc(req))
	})
	if err != nil {
		return err
	}

	req.ID = id

	return nil
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
