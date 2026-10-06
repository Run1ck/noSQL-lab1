package cartapi

import (
	"booking/internal/api/apitest"
	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"booking/internal/fake"
	"booking/internal/httpx"
	"booking/internal/storage/redisstore"
	"booking/internal/usecase"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// Правила корзины проверяют тесты use case'а; здесь — HTTP: разбор запроса,
// коды ошибок, JSON и авторизация.

const cartTTL = 30 * time.Minute

// Даты — от сегодняшнего дня: «сегодня» use case берёт из time.Now().
var (
	yesterday = day(-1)
	tomorrow  = day(1)
	dayAfter  = day(2)
)

func day(offset int) string {
	return service.DateOf(time.Now().AddDate(0, 0, offset)).String()
}

type testEnv struct {
	uc  *usecase.Cart
	mr  *miniredis.Miniredis
	mux *http.ServeMux // от имени Alice
}

// newEnv — use case корзины на miniredis и фейках; room-101 занята послезавтра.
func newEnv(t *testing.T) *testEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	uc := usecase.NewCart(
		redisstore.NewCarts(rdb, cartTTL),
		fake.NewServices(
			&service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
			&service.Service{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: true},
			&service.Service{ID: "old-lab", Name: "Старая лаборатория", Kind: service.Lab, Active: false},
		),
		fake.NewBookings(booking.Booking{ServiceID: "room-101", Date: fake.Date(dayAfter), RequestID: 1}),
	)
	return &testEnv{uc: uc, mr: mr, mux: apitest.Mux(New(uc), apitest.Middlewares(apitest.Alice))}
}

func addBody(serviceID, date string) string {
	return fmt.Sprintf(`{"service_id":%q,"date":%q}`, serviceID, date)
}

// body выполняет запрос, проверяет статус и отдаёт тело без перевода строки.
func body(t *testing.T, mux http.Handler, method, target, reqBody string, wantStatus int) string {
	t.Helper()
	rec := apitest.Do(mux, method, target, reqBody)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s: code = %d, want %d: %s", method, target, rec.Code, wantStatus, rec.Body)
	}
	return strings.TrimSpace(rec.Body.String())
}

const emptyCart = `{"items":[],"expires_in":0}`

func TestCartJSON(t *testing.T) {
	e := newEnv(t)

	if got := body(t, e.mux, http.MethodGet, "/api/cart", "", http.StatusOK); got != emptyCart {
		t.Fatalf("empty cart: %s, want %s", got, emptyCart)
	}

	body(t, e.mux, http.MethodPost, "/api/cart/items", addBody("room-101", tomorrow), http.StatusOK)
	got := body(t, e.mux, http.MethodPost, "/api/cart/items", addBody("lab-1", tomorrow), http.StatusOK)
	want := fmt.Sprintf(`{"items":[{"service_id":"lab-1","date":%q},{"service_id":"room-101","date":%q}],"expires_in":1800}`,
		tomorrow, tomorrow)
	if got != want {
		t.Fatalf("after add:\n got %s\nwant %s", got, want)
	}

	got = body(t, e.mux, http.MethodDelete, "/api/cart/items/room-101/"+tomorrow, "", http.StatusOK)
	want = fmt.Sprintf(`{"items":[{"service_id":"lab-1","date":%q}],"expires_in":1800}`, tomorrow)
	if got != want {
		t.Fatalf("after remove:\n got %s\nwant %s", got, want)
	}

	if got := body(t, e.mux, http.MethodDelete, "/api/cart", "", http.StatusNoContent); got != "" {
		t.Fatalf("clear: body %q, want empty", got)
	}
	if got := body(t, e.mux, http.MethodGet, "/api/cart", "", http.StatusOK); got != emptyCart {
		t.Fatalf("after clear: %s, want %s", got, emptyCart)
	}
}

// expires_in — секунды с округлением вверх: осталось 19 мин 59,5 с → 1200.
func TestExpiresInRoundsUp(t *testing.T) {
	e := newEnv(t)
	body(t, e.mux, http.MethodPost, "/api/cart/items", addBody("room-101", tomorrow), http.StatusOK)

	e.mr.FastForward(10*time.Minute + 500*time.Millisecond)
	if got := body(t, e.mux, http.MethodGet, "/api/cart", "", http.StatusOK); !strings.HasSuffix(got, `"expires_in":1200}`) {
		t.Fatalf("got %s, want expires_in 1200", got)
	}
}

// Каждая ошибка разбора запроса и use case'а — свой статус и код из контракта.
func TestErrorCodes(t *testing.T) {
	e := newEnv(t)

	for _, tc := range []struct {
		method, target, body string
		status               int
		code                 string
	}{
		{http.MethodPost, "/api/cart/items", addBody("room-101", "2026-02-30"), http.StatusBadRequest, "invalid_date"},
		{http.MethodPost, "/api/cart/items", `{"service_id":"room-101"}`, http.StatusBadRequest, "invalid_date"},
		{http.MethodPost, "/api/cart/items", `{"service_id":`, http.StatusBadRequest, "bad_request"},
		{http.MethodPost, "/api/cart/items", `{"service_id":"room-101","date":"` + tomorrow + `","qty":2}`, http.StatusBadRequest, "bad_request"},
		{http.MethodPost, "/api/cart/items", addBody("nope", tomorrow), http.StatusNotFound, "service_not_found"},
		{http.MethodPost, "/api/cart/items", addBody("old-lab", tomorrow), http.StatusUnprocessableEntity, "service_unavailable"},
		{http.MethodPost, "/api/cart/items", addBody("room-101", yesterday), http.StatusUnprocessableEntity, "past_date"},
		{http.MethodPost, "/api/cart/items", addBody("room-101", dayAfter), http.StatusConflict, "slot_booked"},
		{http.MethodDelete, "/api/cart/items/room-101/" + tomorrow, "", http.StatusNotFound, "item_not_found"},
		{http.MethodDelete, "/api/cart/items/room-101/2026-02-30", "", http.StatusBadRequest, "invalid_date"},
	} {
		rec := apitest.Do(e.mux, tc.method, tc.target, tc.body)
		if rec.Code != tc.status || apitest.ErrorCode(rec) != tc.code {
			t.Errorf("%s %s %s: code=%d body=%s, want %d %s", tc.method, tc.target, tc.body,
				rec.Code, rec.Body, tc.status, tc.code)
		}
	}

	e.mr.SetError("ERR boom")
	rec := apitest.Do(e.mux, http.MethodGet, "/api/cart", "")
	if rec.Code != http.StatusInternalServerError || apitest.ErrorCode(rec) != "internal" {
		t.Fatalf("redis down: code=%d body=%s, want 500 internal", rec.Code, rec.Body)
	}
}

func TestRequiresAuth(t *testing.T) {
	e := newEnv(t)
	mux := apitest.Mux(New(e.uc), apitest.RejectAll())

	for _, rq := range [][2]string{
		{http.MethodGet, "/api/cart"},
		{http.MethodPost, "/api/cart/items"},
		{http.MethodDelete, "/api/cart/items/room-101/" + tomorrow},
		{http.MethodDelete, "/api/cart"},
	} {
		if rec := apitest.Do(mux, rq[0], rq[1], ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: code = %d, want 401", rq[0], rq[1], rec.Code)
		}
	}
}

// Маршрут зарегистрировали с Auth, который не кладёт пользователя, — 401, а не
// корзина нулевого UUID.
func TestMissingPrincipal(t *testing.T) {
	e := newEnv(t)
	pass := func(h http.Handler) http.Handler { return h }
	mux := apitest.Mux(New(e.uc), httpx.Middlewares{Auth: pass, Admin: pass})

	rec := apitest.Do(mux, http.MethodGet, "/api/cart", "")
	if rec.Code != http.StatusUnauthorized || apitest.ErrorCode(rec) != "unauthorized" {
		t.Fatalf("code=%d body=%s, want 401 unauthorized", rec.Code, rec.Body)
	}
}

// Корзина — того пользователя, что в контексте.
func TestUsesPrincipalFromContext(t *testing.T) {
	e := newEnv(t)
	body(t, e.mux, http.MethodPost, "/api/cart/items", addBody("room-101", tomorrow), http.StatusOK)

	bobMux := apitest.Mux(New(e.uc), apitest.Middlewares(apitest.Bob))
	if got := body(t, bobMux, http.MethodGet, "/api/cart", "", http.StatusOK); got != emptyCart {
		t.Fatalf("bob sees %s, want empty cart", got)
	}
}
