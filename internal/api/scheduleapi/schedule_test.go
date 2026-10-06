package scheduleapi

import (
	"booking/internal/api/apitest"
	"booking/internal/domain/booking"
	"booking/internal/fake"
	"booking/internal/usecase"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// Правила расписания проверяют тесты use case'а; здесь — HTTP: параметры,
// коды ошибок и JSON.

func newMux(bookings *fake.Bookings) *http.ServeMux {
	return apitest.Mux(New(usecase.NewSchedule(bookings)), apitest.Middlewares(apitest.Alice))
}

func TestGet(t *testing.T) {
	mux := newMux(fake.NewBookings(
		booking.Booking{ServiceID: "room-101", Date: fake.Date("2026-10-05"), RequestID: 1},
		booking.Booking{ServiceID: "lab-1", Date: fake.Date("2026-10-05"), RequestID: 2},
	))

	rec := apitest.Do(mux, http.MethodGet, "/api/schedule?from=2026-10-05&to=2026-10-06", "")
	// У свободного дня "booked":[], не null.
	want := `{"days":[{"date":"2026-10-05","booked":["lab-1","room-101"]},{"date":"2026-10-06","booked":[]}]}`
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != want {
		t.Fatalf("code=%d body:\n got %s\nwant %s", rec.Code, got, want)
	}
}

func TestGet_Errors(t *testing.T) {
	mux := newMux(fake.NewBookings())

	for _, tc := range []struct {
		query  string
		status int
		code   string
	}{
		{"from=2026-02-30&to=2026-03-01", http.StatusBadRequest, "invalid_date"},
		{"from=2026-10-05&to=05.10.2026", http.StatusBadRequest, "invalid_date"},
		{"from=2026-10-05", http.StatusBadRequest, "invalid_date"},
		{"from=2026-10-07&to=2026-10-05", http.StatusBadRequest, "bad_request"},
		{"from=2026-10-01&to=2026-11-01", http.StatusBadRequest, "bad_request"}, // 32 дня
	} {
		rec := apitest.Do(mux, http.MethodGet, "/api/schedule?"+tc.query, "")
		if rec.Code != tc.status || apitest.ErrorCode(rec) != tc.code {
			t.Errorf("%q: code=%d body=%s, want %d %s", tc.query, rec.Code, rec.Body, tc.status, tc.code)
		}
	}
}

func TestGet_RequiresAuth(t *testing.T) {
	mux := apitest.Mux(New(usecase.NewSchedule(fake.NewBookings())), apitest.RejectAll())

	rec := apitest.Do(mux, http.MethodGet, "/api/schedule?from=2026-10-05&to=2026-10-05", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestGet_RepositoryError(t *testing.T) {
	bookings := fake.NewBookings()
	bookings.Err = errors.New("db down")

	rec := apitest.Do(newMux(bookings), http.MethodGet, "/api/schedule?from=2026-10-05&to=2026-10-05", "")
	if rec.Code != http.StatusInternalServerError || apitest.ErrorCode(rec) != "internal" {
		t.Fatalf("code=%d body=%s, want 500 internal", rec.Code, rec.Body)
	}
}
