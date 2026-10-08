package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

const (
	testBanLimit  = 3
	testBanWindow = 10 * time.Minute
	testBanTTL    = 15 * time.Minute
)

// firstInWindow — решение по первому действию в новом окне.
var firstInWindow = Decision{Allowed: true, Limit: testBanLimit, Remaining: testBanLimit - 1}

func newTestBan(t *testing.T) (*Ban, *miniredis.Miniredis) {
	t.Helper()
	rdb, mr := newTestRedis(t)
	return NewBan(rdb, testBanLimit, testBanWindow, testBanTTL), mr
}

// exhaust пропускает testBanLimit действий и проверяет, что следующее ставит бан.
func exhaust(t *testing.T, b *Ban, key string) {
	t.Helper()
	for range testBanLimit {
		require.True(t, mustAllow(t, b, key).Allowed, "within limit")
	}
	require.False(t, mustAllow(t, b, key).Allowed, "over limit")
}

func TestBan_BansOverLimit(t *testing.T) {
	b, mr := newTestBan(t)

	for i := 1; i <= testBanLimit; i++ {
		require.Equal(t, Decision{Allowed: true, Limit: testBanLimit, Remaining: testBanLimit - i},
			mustAllow(t, b, "user"), "action %d", i)
	}

	require.Equal(t, Decision{Limit: testBanLimit, RetryAfter: testBanTTL}, mustAllow(t, b, "user"))
	require.Equal(t, testBanTTL, mr.TTL("ban:user"))
	require.False(t, mr.Exists("rl:req:user"), "counter must be reset on ban")
}

func TestBan_RejectsWhileBanned(t *testing.T) {
	b, mr := newTestBan(t)
	exhaust(t, b, "user")

	mr.FastForward(5 * time.Minute)
	require.Equal(t, Decision{Limit: testBanLimit, RetryAfter: testBanTTL - 5*time.Minute}, mustAllow(t, b, "user"))
	require.False(t, mr.Exists("rl:req:user"), "actions while banned must not be counted")
}

func TestBan_AllowsAfterBanExpires(t *testing.T) {
	b, mr := newTestBan(t)
	exhaust(t, b, "user")

	mr.FastForward(testBanTTL)
	require.Equal(t, firstInWindow, mustAllow(t, b, "user"))
}

func TestBan_WindowExpiresWithoutBan(t *testing.T) {
	b, mr := newTestBan(t)
	for range testBanLimit {
		mustAllow(t, b, "user")
	}

	mr.FastForward(testBanWindow)
	require.Equal(t, firstInWindow, mustAllow(t, b, "user"))
	require.False(t, mr.Exists("ban:user"), "no ban expected")
}

// Окно счётчика отсчитывается от первой заявки и не сдвигается следующими.
func TestBan_WindowDoesNotSlide(t *testing.T) {
	b, mr := newTestBan(t)

	mustAllow(t, b, "user")
	mustAllow(t, b, "user")
	mr.FastForward(testBanWindow - time.Minute)
	require.Equal(t, Decision{Allowed: true, Limit: testBanLimit}, mustAllow(t, b, "user"), "last in window")
	mr.FastForward(time.Minute)

	require.Equal(t, firstInWindow, mustAllow(t, b, "user"), "new window")
}

func TestBan_UsersAreIndependent(t *testing.T) {
	b, _ := newTestBan(t)
	exhaust(t, b, "user")

	require.True(t, mustAllow(t, b, "other").Allowed, "other user must not be banned")
}

func TestBan_Concurrent(t *testing.T) {
	rdb, mr := newTestRedis(t)
	b := NewBan(rdb, 10, testBanWindow, testBanTTL)

	require.Equal(t, 10, allowConcurrently(t, b, "user", 100))
	require.True(t, mr.Exists("ban:user"), "user must be banned")
}

func TestBan_RedisError(t *testing.T) {
	b, mr := newTestBan(t)
	mr.SetError("ERR boom")

	_, err := b.Allow(context.Background(), "user")
	require.Error(t, err)
}
