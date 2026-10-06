package usecase

import (
	"booking/internal/domain/booking"
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/fake"
	"booking/internal/storage/redisstore"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const cartTTL = 30 * time.Minute

var (
	alice = uuid.MustParse("8f1c2a4e-0b7d-4c3e-9a51-6d2f0e8b7c11")
	bob   = uuid.MustParse("3b9e7d10-5c2a-4f8e-b6d4-1a0c9e2f7b33")
)

type cartEnv struct {
	uc       *Cart
	services *fake.Services
	bookings *fake.Bookings
	mr       *miniredis.Miniredis
}

// newCartEnv — настоящие корзины на miniredis, фейки услуг и броней, часы
// стоят на 2026-10-05 12:00.
func newCartEnv(t *testing.T, bookings ...booking.Booking) *cartEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	e := &cartEnv{
		services: fake.NewServices(
			&service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true},
			&service.Service{ID: "lab-1", Name: "Лаборатория", Kind: service.Lab, Active: true},
			&service.Service{ID: "old-lab", Name: "Старая лаборатория", Kind: service.Lab, Active: false},
		),
		bookings: fake.NewBookings(bookings...),
		mr:       mr,
	}
	e.uc = NewCart(redisstore.NewCarts(rdb, cartTTL), e.services, e.bookings)
	e.uc.now = func() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.Local) }
	return e
}

func item(serviceID, date string) cart.Item {
	return cart.Item{ServiceID: serviceID, Date: fake.Date(date)}
}

func (e *cartEnv) mustAdd(t *testing.T, userID uuid.UUID, it cart.Item) CartView {
	t.Helper()
	v, err := e.uc.AddItem(context.Background(), userID, it)
	if err != nil {
		t.Fatalf("AddItem(%+v): %v", it, err)
	}
	return v
}

func (e *cartEnv) mustGet(t *testing.T, userID uuid.UUID) CartView {
	t.Helper()
	v, err := e.uc.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return v
}

func TestCart_GetEmpty(t *testing.T) {
	e := newCartEnv(t)

	if v := e.mustGet(t, alice); len(v.Items) != 0 || v.ExpiresIn != 0 {
		t.Fatalf("got %+v, want empty cart with ExpiresIn 0", v)
	}
}

func TestCart_AddItem(t *testing.T) {
	e := newCartEnv(t)

	v := e.mustAdd(t, alice, item("room-101", "2026-10-06"))
	if !slices.Equal(v.Items, []cart.Item{item("room-101", "2026-10-06")}) || v.ExpiresIn != cartTTL {
		t.Fatalf("got %+v, want one item and ExpiresIn %s", v, cartTTL)
	}

	e.mustAdd(t, alice, item("room-101", "2026-10-07"))
	e.mustAdd(t, alice, item("lab-1", "2026-10-06"))
	e.mustAdd(t, alice, item("lab-1", "2026-10-05")) // сегодня — можно
	// По дате, затем по ID услуги.
	want := []cart.Item{
		item("lab-1", "2026-10-05"),
		item("lab-1", "2026-10-06"),
		item("room-101", "2026-10-06"),
		item("room-101", "2026-10-07"),
	}
	if v := e.mustGet(t, alice); !slices.Equal(v.Items, want) {
		t.Fatalf("items:\n got %v\nwant %v", v.Items, want)
	}
}

// Повторное добавление — не ошибка: позиция одна, TTL продлён.
func TestCart_AddItemTwiceProlongs(t *testing.T) {
	e := newCartEnv(t)

	e.mustAdd(t, alice, item("room-101", "2026-10-06"))
	e.mr.FastForward(10 * time.Minute)
	if v := e.mustAdd(t, alice, item("room-101", "2026-10-06")); len(v.Items) != 1 || v.ExpiresIn != cartTTL {
		t.Fatalf("got %+v, want 1 item and ExpiresIn %s", v, cartTTL)
	}
}

func TestCart_AddItemRules(t *testing.T) {
	e := newCartEnv(t, booking.Booking{ServiceID: "room-101", Date: fake.Date("2026-10-07"), RequestID: 1})

	for _, tc := range []struct {
		name string
		item cart.Item
		want error
	}{
		{"unknown service", item("nope", "2026-10-06"), service.ErrNotFound},
		{"inactive service", item("old-lab", "2026-10-06"), cart.ErrServiceUnavailable},
		{"past date", item("room-101", "2026-10-04"), cart.ErrPastDate},
		{"booked slot", item("room-101", "2026-10-07"), cart.ErrSlotBooked},
		// Порядок проверок: неактивная услуга важнее прошедшей даты.
		{"inactive and past", item("old-lab", "2026-10-04"), cart.ErrServiceUnavailable},
	} {
		if _, err := e.uc.AddItem(context.Background(), alice, tc.item); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
	// Ни одна ошибка ничего не положила в корзину.
	if v := e.mustGet(t, alice); len(v.Items) != 0 {
		t.Fatalf("cart after errors: %v, want empty", v.Items)
	}
	// Бронь другой услуги на тот же день не мешает.
	e.mustAdd(t, alice, item("lab-1", "2026-10-07"))
}

// «Сегодня» — по часам use case'а, а не по реальному времени.
func TestCart_TodayFromClock(t *testing.T) {
	e := newCartEnv(t)
	e.uc.now = func() time.Time { return time.Date(2026, 10, 5, 23, 59, 0, 0, time.Local) }

	e.mustAdd(t, alice, item("room-101", "2026-10-05"))
	if _, err := e.uc.AddItem(context.Background(), alice, item("room-101", "2026-10-04")); !errors.Is(err, cart.ErrPastDate) {
		t.Fatalf("err = %v, want ErrPastDate", err)
	}
}

func TestCart_RemoveItem(t *testing.T) {
	e := newCartEnv(t)
	e.mustAdd(t, alice, item("room-101", "2026-10-06"))
	e.mustAdd(t, alice, item("lab-1", "2026-10-06"))

	v, err := e.uc.RemoveItem(context.Background(), alice, item("room-101", "2026-10-06"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(v.Items, []cart.Item{item("lab-1", "2026-10-06")}) {
		t.Fatalf("items = %v, want only lab-1", v.Items)
	}
	if _, err := e.uc.RemoveItem(context.Background(), alice, item("room-101", "2026-10-06")); !errors.Is(err, cart.ErrItemNotFound) {
		t.Fatalf("second remove: err = %v, want ErrItemNotFound", err)
	}
}

func TestCart_Clear(t *testing.T) {
	e := newCartEnv(t)
	e.mustAdd(t, alice, item("room-101", "2026-10-06"))

	for range 2 { // очистка пустой корзины — тоже не ошибка
		if err := e.uc.Clear(context.Background(), alice); err != nil {
			t.Fatal(err)
		}
	}
	if v := e.mustGet(t, alice); len(v.Items) != 0 || v.ExpiresIn != 0 {
		t.Fatalf("after clear: %+v, want empty", v)
	}
}

func TestCart_ExpiresWithTTL(t *testing.T) {
	e := newCartEnv(t)
	e.mustAdd(t, alice, item("room-101", "2026-10-06"))

	e.mr.FastForward(10 * time.Minute)
	if v := e.mustGet(t, alice); v.ExpiresIn != 20*time.Minute {
		t.Fatalf("ExpiresIn = %s, want 20m", v.ExpiresIn)
	}
	// Корзина истекла — она просто пустая, а не ошибка.
	e.mr.FastForward(20 * time.Minute)
	if v := e.mustGet(t, alice); len(v.Items) != 0 || v.ExpiresIn != 0 {
		t.Fatalf("expired cart: %+v, want empty", v)
	}
}

func TestCart_PerUser(t *testing.T) {
	e := newCartEnv(t)
	e.mustAdd(t, alice, item("room-101", "2026-10-06"))

	if v := e.mustGet(t, bob); len(v.Items) != 0 {
		t.Fatalf("bob sees %v, want empty cart", v.Items)
	}
}

// Отказ хранилища — ошибка, а не доменный ответ вроде «услуги нет».
func TestCart_StorageErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail func(e *cartEnv)
	}{
		{"services down", func(e *cartEnv) { e.services.Err = errors.New("db down") }},
		{"bookings down", func(e *cartEnv) { e.bookings.Err = errors.New("db down") }},
		{"redis down", func(e *cartEnv) { e.mr.SetError("ERR boom") }},
	} {
		e := newCartEnv(t)
		tc.fail(e)
		_, err := e.uc.AddItem(context.Background(), alice, item("room-101", "2026-10-06"))
		if err == nil || errors.Is(err, service.ErrNotFound) || errors.Is(err, cart.ErrSlotBooked) {
			t.Errorf("%s: err = %v, want storage error", tc.name, err)
		}
	}
}
