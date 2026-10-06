package usecase

import (
	"booking/internal/domain/service"
	"booking/internal/fake"
	"context"
	"errors"
	"slices"
	"testing"
)

func TestCatalog_Active(t *testing.T) {
	uc := NewCatalog(fake.NewServices(
		&service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
		&service.Service{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
		&service.Service{ID: "proj-1", Name: "Проектор", Kind: service.Equipment, Active: true},
	))

	got, err := uc.Active(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, s := range got {
		ids = append(ids, s.ID)
	}
	// Только активные, в порядке List.
	if want := []string{"room-101", "proj-1"}; !slices.Equal(ids, want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
}

func TestCatalog_RepositoryError(t *testing.T) {
	services := fake.NewServices()
	services.Err = errors.New("db down")

	if _, err := NewCatalog(services).Active(context.Background()); !errors.Is(err, services.Err) {
		t.Fatalf("err = %v, want repository error", err)
	}
}
