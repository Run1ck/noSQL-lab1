package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

const (
	testBanLimit  = 3
	testBanWindow = 10 * time.Minute
	testBanTTL    = 15 * time.Minute
)

func newTestBan(t *testing.T) (*Ban, *miniredis.Miniredis) {
	t.Helper()
	rdb, mr := newTestRedis(t)
	return NewBan(rdb, testBanLimit, testBanWindow, testBanTTL), mr
}

// exhaust пропускает testBanLimit действий и проверяет, что следующее ставит бан.
func exhaust(t *testing.T, b *Ban, key string) {
	t.Helper()
	for range testBanLimit {
		if d := mustAllow(t, b, key); !d.Allowed {
			t.Fatalf("within limit: got %+v, want allowed", d)
		}
	}
	if d := mustAllow(t, b, key); d.Allowed {
		t.Fatalf("over limit: got %+v, want banned", d)
	}
}

func TestBan_BansOverLimit(t *testing.T) {
	b, mr := newTestBan(t)

	for i := 1; i <= testBanLimit; i++ {
		d := mustAllow(t, b, "user")
		want := Decision{Allowed: true, Limit: testBanLimit, Remaining: testBanLimit - i}
		if d != want {
			t.Fatalf("action %d: got %+v, want %+v", i, d, want)
		}
	}

	d := mustAllow(t, b, "user")
	want := Decision{Limit: testBanLimit, RetryAfter: testBanTTL}
	if d != want {
		t.Fatalf("over limit: got %+v, want %+v", d, want)
	}
	if got := mr.TTL("ban:user"); got != testBanTTL {
		t.Fatalf("ban TTL = %s, want %s", got, testBanTTL)
	}
	if mr.Exists("rl:req:user") {
		t.Fatal("counter must be reset on ban")
	}
}

func TestBan_RejectsWhileBanned(t *testing.T) {
	b, mr := newTestBan(t)
	exhaust(t, b, "user")

	mr.FastForward(5 * time.Minute)
	d := mustAllow(t, b, "user")
	if d.Allowed || d.RetryAfter != testBanTTL-5*time.Minute {
		t.Fatalf("while banned: got %+v, want rejected with RetryAfter %s", d, testBanTTL-5*time.Minute)
	}
	if mr.Exists("rl:req:user") {
		t.Fatal("actions while banned must not be counted")
	}
}

func TestBan_AllowsAfterBanExpires(t *testing.T) {
	b, mr := newTestBan(t)
	exhaust(t, b, "user")

	mr.FastForward(testBanTTL)
	d := mustAllow(t, b, "user")
	if !d.Allowed || d.Remaining != testBanLimit-1 {
		t.Fatalf("after ban: got %+v, want allowed with %d remaining", d, testBanLimit-1)
	}
}

func TestBan_WindowExpiresWithoutBan(t *testing.T) {
	b, mr := newTestBan(t)
	for range testBanLimit {
		mustAllow(t, b, "user")
	}

	mr.FastForward(testBanWindow)
	d := mustAllow(t, b, "user")
	if !d.Allowed || d.Remaining != testBanLimit-1 {
		t.Fatalf("new window: got %+v, want allowed with %d remaining", d, testBanLimit-1)
	}
	if mr.Exists("ban:user") {
		t.Fatal("no ban expected")
	}
}

// Окно счётчика отсчитывается от первой заявки и не сдвигается следующими.
func TestBan_WindowDoesNotSlide(t *testing.T) {
	b, mr := newTestBan(t)

	mustAllow(t, b, "user")
	mustAllow(t, b, "user")
	mr.FastForward(testBanWindow - time.Minute)
	if d := mustAllow(t, b, "user"); !d.Allowed || d.Remaining != 0 {
		t.Fatalf("last in window: got %+v, want allowed with 0 remaining", d)
	}
	mr.FastForward(time.Minute)

	d := mustAllow(t, b, "user")
	if !d.Allowed || d.Remaining != testBanLimit-1 {
		t.Fatalf("new window: got %+v, want allowed with %d remaining", d, testBanLimit-1)
	}
}

// Счётчик без TTL получает TTL окна, а не копится вечно.
func TestBan_HealsCounterWithoutTTL(t *testing.T) {
	b, mr := newTestBan(t)
	if err := mr.Set("rl:req:user", "1"); err != nil {
		t.Fatal(err)
	}

	if d := mustAllow(t, b, "user"); !d.Allowed || d.Remaining != testBanLimit-2 {
		t.Fatalf("got %+v, want allowed with %d remaining", d, testBanLimit-2)
	}
	if got := mr.TTL("rl:req:user"); got != testBanWindow {
		t.Fatalf("counter TTL = %s, want %s", got, testBanWindow)
	}
}

// Длительности меньше миллисекунды: бан всё равно ставится (SET … PX 0 — ошибка).
func TestBan_SubMillisecondDurations(t *testing.T) {
	rdb, mr := newTestRedis(t)
	b := NewBan(rdb, 1, 500*time.Microsecond, 500*time.Microsecond)

	if d := mustAllow(t, b, "user"); !d.Allowed {
		t.Fatalf("first: got %+v, want allowed", d)
	}
	d := mustAllow(t, b, "user")
	if d.Allowed || d.RetryAfter != time.Millisecond {
		t.Fatalf("second: got %+v, want banned with RetryAfter 1ms", d)
	}
	if got := mr.TTL("ban:user"); got != time.Millisecond {
		t.Fatalf("ban TTL = %s, want 1ms", got)
	}
}

func TestBan_UsersAreIndependent(t *testing.T) {
	b, _ := newTestBan(t)
	exhaust(t, b, "user")

	if d := mustAllow(t, b, "other"); !d.Allowed {
		t.Fatalf("other user must not be banned, got %+v", d)
	}
}

// Бан без TTL не должен висеть вечно.
func TestBan_HealsBanWithoutTTL(t *testing.T) {
	b, mr := newTestBan(t)
	if err := mr.Set("ban:user", "1"); err != nil {
		t.Fatal(err)
	}

	d := mustAllow(t, b, "user")
	if d.Allowed || d.RetryAfter != testBanTTL {
		t.Fatalf("got %+v, want rejected with RetryAfter %s", d, testBanTTL)
	}
	if got := mr.TTL("ban:user"); got != testBanTTL {
		t.Fatalf("ban TTL = %s, want %s", got, testBanTTL)
	}
}

func TestBan_Concurrent(t *testing.T) {
	rdb, mr := newTestRedis(t)
	b := NewBan(rdb, 10, testBanWindow, testBanTTL)

	if got := allowConcurrently(t, b, "user", 100); got != 10 {
		t.Fatalf("allowed %d of 100 concurrent actions, want 10", got)
	}
	if !mr.Exists("ban:user") {
		t.Fatal("user must be banned")
	}
}

func TestBan_RedisError(t *testing.T) {
	b, mr := newTestBan(t)
	mr.SetError("ERR boom")

	if _, err := b.Allow(context.Background(), "user"); err == nil {
		t.Fatal("want error")
	}
}
