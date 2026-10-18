package firestore

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"booking/internal/domain/request"
	"booking/internal/domain/service"
)

const requestSchemaVersion = 2

type requestDoc struct {
	UserID        string      `firestore:"userId"`
	Items         []itemDoc   `firestore:"items"`
	ServiceIDs    []string    `firestore:"serviceIds"`
	CreatedAt     time.Time   `firestore:"createdAt"`
	Status        statusDoc   `firestore:"status"`
	Details       *detailsDoc `firestore:"details,omitempty"`
	SchemaVersion int         `firestore:"schemaVersion"`
}

type itemDoc struct {
	ServiceID string `firestore:"serviceId"`
	Date      string `firestore:"date"`
}

type statusDoc struct {
	Code      string    `firestore:"code"`
	Title     string    `firestore:"title"`
	ChangedAt time.Time `firestore:"changedAt"`
	ChangedBy string    `firestore:"changedBy"`
	Comment   string    `firestore:"comment"`
}

type detailsDoc struct {
	Purpose        string `firestore:"purpose"`
	Participants   int    `firestore:"participants"`
	NeedsProjector bool   `firestore:"needsProjector"`
}

func newRequestDoc(req *request.Request) requestDoc {
	doc := requestDoc{
		UserID:        req.UserID.String(),
		Items:         make([]itemDoc, 0, len(req.Items)),
		ServiceIDs:    make([]string, 0, len(req.Items)),
		CreatedAt:     req.CreatedAt,
		Status:        newStatusDoc(req),
		SchemaVersion: requestSchemaVersion,
	}
	for _, item := range req.Items {
		doc.Items = append(doc.Items, itemDoc{ServiceID: item.ServiceID, Date: item.Date.String()})
		if !slices.Contains(doc.ServiceIDs, item.ServiceID) {
			doc.ServiceIDs = append(doc.ServiceIDs, item.ServiceID)
		}
	}
	if req.Details != nil {
		doc.Details = &detailsDoc{
			Purpose:        req.Details.Purpose,
			Participants:   req.Details.Participants,
			NeedsProjector: req.Details.NeedsProjector,
		}
	}

	return doc
}

func newStatusDoc(req *request.Request) statusDoc {
	status := statusDoc{
		Code:      string(req.Status),
		ChangedAt: req.CreatedAt,
		ChangedBy: req.UserID.String(),
		Comment:   req.Comment,
	}
	if req.ProcessedAt != nil {
		status.ChangedAt = *req.ProcessedAt
	}
	if req.ProcessedBy != nil {
		status.ChangedBy = req.ProcessedBy.String()
	}

	return status
}

func (d requestDoc) toDomain(id int64) (*request.Request, error) {
	userID, err := uuid.Parse(d.UserID)
	if err != nil {
		return nil, fmt.Errorf("request %d: user id: %w", id, err)
	}

	req := &request.Request{
		ID:        id,
		UserID:    userID,
		Items:     make([]request.Item, 0, len(d.Items)),
		Status:    request.Status(d.Status.Code),
		Comment:   d.Status.Comment,
		CreatedAt: d.CreatedAt,
	}
	for _, item := range d.Items {
		date, err := service.ParseDate(item.Date)
		if err != nil {
			return nil, fmt.Errorf("request %d: item date: %w", id, err)
		}
		req.Items = append(req.Items, request.Item{ServiceID: item.ServiceID, Date: date})
	}
	if d.Details != nil {
		req.Details = &request.Details{
			Purpose:        d.Details.Purpose,
			Participants:   d.Details.Participants,
			NeedsProjector: d.Details.NeedsProjector,
		}
	}
	if req.Status != request.StatusNew {
		at := d.Status.ChangedAt
		req.ProcessedAt = &at

		by, err := uuid.Parse(d.Status.ChangedBy)
		if err != nil {
			return nil, fmt.Errorf("request %d: changed by: %w", id, err)
		}
		req.ProcessedBy = &by
	}

	return req, nil
}
