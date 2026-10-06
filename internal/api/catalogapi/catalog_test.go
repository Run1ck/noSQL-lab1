package catalogapi

import (
	"booking/internal/api/apitest"
	"booking/internal/domain/service"
	"booking/internal/fake"
	"booking/internal/usecase"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func newMux(services *fake.Services) *http.ServeMux {
	// RejectAll: каталог открыт всем, авторизация не нужна.
	return apitest.Mux(New(usecase.NewCatalog(services)), apitest.RejectAll())
}

func TestList(t *testing.T) {
	mux := newMux(fake.NewServices(
		&service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
		&service.Service{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: false},
	))

	rec := apitest.Do(mux, http.MethodGet, "/api/services", "")
	want := `[{"id":"room-101","name":"Аудитория 101","kind":"room","active":true}]`
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != want {
		t.Fatalf("code=%d body:\n got %s\nwant %s", rec.Code, got, want)
	}
}

func TestList_Empty(t *testing.T) {
	rec := apitest.Do(newMux(fake.NewServices()), http.MethodGet, "/api/services", "")
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != "[]" {
		t.Fatalf("code=%d body=%s, want 200 []", rec.Code, got)
	}
}

func TestList_RepositoryError(t *testing.T) {
	services := fake.NewServices()
	services.Err = errors.New("db down")

	rec := apitest.Do(newMux(services), http.MethodGet, "/api/services", "")
	if rec.Code != http.StatusInternalServerError || apitest.ErrorCode(rec) != "internal" {
		t.Fatalf("code=%d body=%s, want 500 internal", rec.Code, rec.Body)
	}
}
