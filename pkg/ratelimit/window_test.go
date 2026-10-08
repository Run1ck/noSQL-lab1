package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testLimit  = 3
	testWindow = time.Minute
)

func TestFixedWindow_LimitsWithinWindow(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for i := 1; i <= testLimit; i++ {
		require.Equal(t, Decision{Allowed: true, Limit: testLimit, Remaining: testLimit - i},
			mustAllow(t, l, "1.2.3.4"), "action %d", i)
	}

	mr.FastForward(20 * time.Second)
	require.Equal(t, Decision{Limit: testLimit, RetryAfter: 40 * time.Second}, mustAllow(t, l, "1.2.3.4"))
	require.Equal(t, 40*time.Second, mr.TTL("rl:api:1.2.3.4"))
}

func TestFixedWindow_ResetsAfterWindow(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for range testLimit + 1 {
		mustAllow(t, l, "1.2.3.4")
	}
	mr.FastForward(testWindow)

	require.Equal(t, Decision{Allowed: true, Limit: testLimit, Remaining: testLimit - 1}, mustAllow(t, l, "1.2.3.4"))
}

func TestFixedWindow_KeysAreIndependent(t *testing.T) {
	rdb, _ := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for range testLimit + 1 {
		mustAllow(t, l, "1.2.3.4")
	}
	require.True(t, mustAllow(t, l, "5.6.7.8").Allowed, "other key must not be limited")
}

// Счётчик без TTL (например, после ручного INCR) не должен блокировать навсегда.
func TestFixedWindow_HealsCounterWithoutTTL(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)
	require.NoError(t, mr.Set("rl:api:1.2.3.4", "100"))

	require.Equal(t, Decision{Limit: testLimit, RetryAfter: testWindow}, mustAllow(t, l, "1.2.3.4"))
	mr.FastForward(testWindow)
	require.True(t, mustAllow(t, l, "1.2.3.4").Allowed, "after window")
}

func TestFixedWindow_Concurrent(t *testing.T) {
	rdb, _ := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", 10, testWindow)

	require.Equal(t, 10, allowConcurrently(t, l, "1.2.3.4", 100))
}

func TestFixedWindow_RedisError(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)
	mr.SetError("ERR boom")

	_, err := l.Allow(context.Background(), "1.2.3.4")
	require.Error(t, err)
}
