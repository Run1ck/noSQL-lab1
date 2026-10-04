package redisstore

import (
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const testTTL = 30 * time.Minute

func newTestCarts(t *testing.T) (*Carts, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewCarts(rdb, testTTL), mr
}

func item(serviceID, date string) cart.Item {
	d, err := service.ParseDate(date)
	if err != nil {
		panic(err)
	}
	return cart.Item{ServiceID: serviceID, Date: d}
}

func cartWith(userID uuid.UUID, items ...cart.Item) *cart.Cart {
	c := &cart.Cart{UserID: userID, Items: make(map[cart.Item]struct{})}
	for _, it := range items {
		c.AddItem(it)
	}
	return c
}

func mustGet(t *testing.T, r *Carts, userID uuid.UUID) *cart.Cart {
	t.Helper()
	c, err := r.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return c
}

func mustTTL(t *testing.T, r *Carts, userID uuid.UUID) time.Duration {
	t.Helper()
	d, err := r.TTL(context.Background(), userID)
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	return d
}

func TestCarts_GetMissingReturnsEmpty(t *testing.T) {
	r, _ := newTestCarts(t)
	userID := uuid.New()

	c := mustGet(t, r, userID)
	if c.UserID != userID || !c.IsEmpty() || c.Items == nil {
		t.Fatalf("want empty cart of %s, got %+v", userID, c)
	}
	if d := mustTTL(t, r, userID); d != 0 {
		t.Fatalf("TTL of missing cart = %s, want 0", d)
	}
}

func TestCarts_SaveAndGet(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("lab|odd-id", "2026-10-06")

	if err := r.Save(ctx, cartWith(userID, a, b)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	c := mustGet(t, r, userID)
	if len(c.Items) != 2 {
		t.Fatalf("want 2 items, got %v", c.Items)
	}
	for _, it := range []cart.Item{a, b} {
		if _, ok := c.Items[it]; !ok {
			t.Errorf("item %+v lost", it)
		}
	}
	if got := mr.TTL(cartKey(userID)); got != testTTL {
		t.Fatalf("key TTL = %s, want %s", got, testTTL)
	}
	if got := mr.Type(cartKey(userID)); got != "set" {
		t.Fatalf("key type = %q, want set", got)
	}
}

func TestCarts_SaveReplacesAndProlongs(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("room-102", "2026-10-05")

	if err := r.Save(ctx, cartWith(userID, a)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	mr.FastForward(10 * time.Minute)
	if err := r.Save(ctx, cartWith(userID, b)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	c := mustGet(t, r, userID)
	if _, ok := c.Items[b]; !ok || len(c.Items) != 1 {
		t.Fatalf("want only %+v, got %v", b, c.Items)
	}
	if d := mustTTL(t, r, userID); d != testTTL {
		t.Fatalf("TTL after Save = %s, want %s", d, testTTL)
	}
}

func TestCarts_SaveEmptyDeletes(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	if err := r.Save(ctx, cartWith(userID, item("room-101", "2026-10-05"))); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.Save(ctx, cartWith(userID)); err != nil {
		t.Fatalf("Save empty: %v", err)
	}
	if mr.Exists(cartKey(userID)) {
		t.Fatal("empty cart must be deleted")
	}
}

func TestCarts_RemoveItem(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("room-102", "2026-10-05")

	if err := r.Save(ctx, cartWith(userID, a, b)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	mr.FastForward(10 * time.Minute)

	if err := r.RemoveItem(ctx, userID, a); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	c := mustGet(t, r, userID)
	if _, ok := c.Items[b]; !ok || len(c.Items) != 1 {
		t.Fatalf("want only %+v, got %v", b, c.Items)
	}
	if d := mustTTL(t, r, userID); d != testTTL {
		t.Fatalf("TTL after RemoveItem = %s, want %s", d, testTTL)
	}

	// Последняя позиция: SET пустеет и пропадает вместе с ключом.
	if err := r.RemoveItem(ctx, userID, b); err != nil {
		t.Fatalf("RemoveItem last: %v", err)
	}
	if mr.Exists(cartKey(userID)) {
		t.Fatal("cart without items must be gone")
	}
}

func TestCarts_RemoveMissingItem(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	if err := r.RemoveItem(ctx, userID, item("room-101", "2026-10-05")); !errors.Is(err, cart.ErrItemNotFound) {
		t.Fatalf("RemoveItem on missing cart: err = %v, want ErrItemNotFound", err)
	}

	if err := r.Save(ctx, cartWith(userID, item("room-101", "2026-10-05"))); err != nil {
		t.Fatalf("Save: %v", err)
	}
	mr.FastForward(10 * time.Minute)
	if err := r.RemoveItem(ctx, userID, item("room-101", "2026-10-06")); !errors.Is(err, cart.ErrItemNotFound) {
		t.Fatalf("RemoveItem of missing item: err = %v, want ErrItemNotFound", err)
	}
	// Неудачное удаление не должно продлевать корзину.
	if d := mustTTL(t, r, userID); d != testTTL-10*time.Minute {
		t.Fatalf("TTL = %s, want %s", d, testTTL-10*time.Minute)
	}
}

func TestCarts_Expires(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	if err := r.Save(ctx, cartWith(userID, item("room-101", "2026-10-05"))); err != nil {
		t.Fatalf("Save: %v", err)
	}
	mr.FastForward(testTTL)

	if c := mustGet(t, r, userID); !c.IsEmpty() {
		t.Fatalf("expired cart must be empty, got %v", c.Items)
	}
	if d := mustTTL(t, r, userID); d != 0 {
		t.Fatalf("TTL of expired cart = %s, want 0", d)
	}
}

func TestCarts_DeleteCart(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	if err := r.DeleteCart(ctx, userID); err != nil {
		t.Fatalf("DeleteCart on missing cart: %v", err)
	}
	if err := r.Save(ctx, cartWith(userID, item("room-101", "2026-10-05"))); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := r.DeleteCart(ctx, userID); err != nil {
		t.Fatalf("DeleteCart: %v", err)
	}
	if mr.Exists(cartKey(userID)) {
		t.Fatal("cart must be deleted")
	}
}

func TestCarts_GetCorruptedMember(t *testing.T) {
	for _, member := range []string{"no-separator", "room-101|2026-02-30"} {
		t.Run(member, func(t *testing.T) {
			r, mr := newTestCarts(t)
			userID := uuid.New()
			if _, err := mr.SAdd(cartKey(userID), member); err != nil {
				t.Fatal(err)
			}

			_, err := r.Get(context.Background(), userID)
			if err == nil {
				t.Fatal("want error for corrupted member")
			}
			if errors.Is(err, service.ErrInvalidDate) {
				t.Fatal("corrupted data must not map to invalid_date")
			}
		})
	}
}

func TestCarts_SaveRejectsInvalidItem(t *testing.T) {
	r, mr := newTestCarts(t)
	userID := uuid.New()
	valid := service.Date{Year: 2026, Month: time.October, Day: 5}

	for _, it := range []cart.Item{
		{ServiceID: "room-101"},
		{ServiceID: "room-101", Date: service.Date{Year: 2026, Month: time.February, Day: 30}},
		{Date: valid},
	} {
		if err := r.Save(context.Background(), cartWith(userID, it)); err == nil {
			t.Errorf("Save(%+v): want error", it)
		}
	}
	if mr.Exists(cartKey(userID)) {
		t.Fatal("invalid cart must not be written")
	}
}

// Ключ без TTL корзины не пишут, но TTL() должен отдать 0, а не -1ns.
func TestCarts_TTLOfKeyWithoutExpiry(t *testing.T) {
	r, mr := newTestCarts(t)
	userID := uuid.New()
	if _, err := mr.SAdd(cartKey(userID), "room-101|2026-10-05"); err != nil {
		t.Fatal(err)
	}

	if d := mustTTL(t, r, userID); d != 0 {
		t.Fatalf("TTL = %s, want 0", d)
	}
}

// Ошибка Redis — это ошибка (500), а не пустая корзина или 404.
func TestCarts_RedisErrors(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	mr.SetError("ERR boom")

	if _, err := r.Get(ctx, userID); err == nil {
		t.Error("Get: want error")
	}
	if err := r.Save(ctx, cartWith(userID, item("room-101", "2026-10-05"))); err == nil {
		t.Error("Save: want error")
	}
	if err := r.RemoveItem(ctx, userID, item("room-101", "2026-10-05")); err == nil || errors.Is(err, cart.ErrItemNotFound) {
		t.Errorf("RemoveItem: err = %v, want Redis error", err)
	}
	if err := r.DeleteCart(ctx, userID); err == nil {
		t.Error("DeleteCart: want error")
	}
	if _, err := r.TTL(ctx, userID); err == nil {
		t.Error("TTL: want error")
	}
}
