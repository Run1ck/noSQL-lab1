package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func New(ctx context.Context, projectID string) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	_, err = client.Collection("_health").Doc("ping").Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		_ = client.Close()

		return nil, fmt.Errorf("firestore ping: %w", err)
	}

	return client, nil
}
