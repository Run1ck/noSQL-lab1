package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
)

const testTTL = 30 * time.Minute

func newTestCarts(t *testing.T) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return New(rdb, testTTL), mr
}

func item(serviceID, date string) cart.Item {
	d, err := service.ParseDate(date)
	if err != nil {
		panic(err)
	}
	return cart.Item{ServiceID: serviceID, Date: d}
}

// items — позиции в том виде, в каком их хранит cart.Cart.
func items(its ...cart.Item) map[cart.Item]struct{} {
	m := make(map[cart.Item]struct{}, len(its))
	for _, it := range its {
		m[it] = struct{}{}
	}
	return m
}

func mustAdd(t *testing.T, r *Redis, userID uuid.UUID, its ...cart.Item) {
	t.Helper()
	for _, it := range its {
		require.NoError(t, r.AddCartItem(context.Background(), userID, it))
	}
}

func mustGet(t *testing.T, r *Redis, userID uuid.UUID) *cart.Cart {
	t.Helper()
	c, err := r.GetCart(context.Background(), userID)
	require.NoError(t, err)
	return c
}

func TestCarts_GetMissingReturnsEmpty(t *testing.T) {
	r, _ := newTestCarts(t)
	userID := uuid.New()

	// Items — пустая map, а не nil; TTL — 0.
	require.Equal(t, &cart.Cart{UserID: userID, Items: items()}, mustGet(t, r, userID))
}

func TestCarts_AddAndGet(t *testing.T) {
	r, mr := newTestCarts(t)
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("lab|odd-id", "2026-10-06")

	mustAdd(t, r, userID, a, b)

	require.Equal(t, &cart.Cart{UserID: userID, Items: items(a, b), TTL: testTTL}, mustGet(t, r, userID))
	require.Equal(t, "set", mr.Type(cartKey(userID)))
}

// Любое добавление, и повтор позиции тоже, продлевает всю корзину.
func TestCarts_AddProlongs(t *testing.T) {
	r, mr := newTestCarts(t)
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("room-102", "2026-10-05")

	mustAdd(t, r, userID, a)
	mr.FastForward(10 * time.Minute)
	mustAdd(t, r, userID, b)
	require.Equal(t, &cart.Cart{UserID: userID, Items: items(a, b), TTL: testTTL}, mustGet(t, r, userID))

	mr.FastForward(10 * time.Minute)
	mustAdd(t, r, userID, a)
	require.Equal(t, &cart.Cart{UserID: userID, Items: items(a, b), TTL: testTTL}, mustGet(t, r, userID))
}

func TestCarts_RemoveItem(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	a, b := item("room-101", "2026-10-05"), item("room-102", "2026-10-05")

	mustAdd(t, r, userID, a, b)
	mr.FastForward(10 * time.Minute)

	require.NoError(t, r.RemoveCartItem(ctx, userID, a))
	require.Equal(t, &cart.Cart{UserID: userID, Items: items(b), TTL: testTTL}, mustGet(t, r, userID))

	// Последняя позиция: SET пустеет и пропадает вместе с ключом.
	require.NoError(t, r.RemoveCartItem(ctx, userID, b))
	require.False(t, mr.Exists(cartKey(userID)), "cart without items must be gone")
}

func TestCarts_RemoveMissingItem(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	require.ErrorIs(t, r.RemoveCartItem(ctx, userID, item("room-101", "2026-10-05")), cart.ErrItemNotFound)

	mustAdd(t, r, userID, item("room-101", "2026-10-05"))
	mr.FastForward(10 * time.Minute)
	require.ErrorIs(t, r.RemoveCartItem(ctx, userID, item("room-101", "2026-10-06")), cart.ErrItemNotFound)

	// Неудачное удаление не должно продлевать корзину.
	require.Equal(t, testTTL-10*time.Minute, mustGet(t, r, userID).TTL)
}

func TestCarts_Expires(t *testing.T) {
	r, mr := newTestCarts(t)
	userID := uuid.New()

	mustAdd(t, r, userID, item("room-101", "2026-10-05"))
	mr.FastForward(testTTL)

	require.Equal(t, &cart.Cart{UserID: userID, Items: items()}, mustGet(t, r, userID))
}

func TestCarts_DeleteCart(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()

	require.NoError(t, r.DeleteCart(ctx, userID), "missing cart")

	mustAdd(t, r, userID, item("room-101", "2026-10-05"))
	require.NoError(t, r.DeleteCart(ctx, userID))
	require.False(t, mr.Exists(cartKey(userID)), "cart must be deleted")
}

func TestCarts_GetCorruptedMember(t *testing.T) {
	for _, member := range []string{"no-separator", "room-101|2026-02-30"} {
		t.Run(member, func(t *testing.T) {
			r, mr := newTestCarts(t)
			userID := uuid.New()
			_, err := mr.SAdd(cartKey(userID), member)
			require.NoError(t, err)

			_, err = r.GetCart(context.Background(), userID)
			require.Error(t, err)
			require.NotErrorIs(t, err, service.ErrInvalidDate, "corrupted data must not map to invalid_date")
		})
	}
}

// Ошибка Redis — это ошибка (500), а не пустая корзина или 404.
func TestCarts_RedisErrors(t *testing.T) {
	r, mr := newTestCarts(t)
	ctx := context.Background()
	userID := uuid.New()
	it := item("room-101", "2026-10-05")
	mr.SetError("ERR boom")

	_, err := r.GetCart(ctx, userID)
	require.Error(t, err, "GetCart")
	require.Error(t, r.AddCartItem(ctx, userID, it), "AddCartItem")
	err = r.RemoveCartItem(ctx, userID, it)
	require.Error(t, err, "RemoveCartItem")
	require.NotErrorIs(t, err, cart.ErrItemNotFound, "RemoveCartItem")
	require.Error(t, r.DeleteCart(ctx, userID), "DeleteCart")
}
