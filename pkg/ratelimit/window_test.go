package ratelimit

import (
	"context"
	"testing"
	"time"
)

const (
	testLimit  = 3
	testWindow = time.Minute
)

func TestFixedWindow_LimitsWithinWindow(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for i := 1; i <= testLimit; i++ {
		d := mustAllow(t, l, "1.2.3.4")
		want := Decision{Allowed: true, Limit: testLimit, Remaining: testLimit - i}
		if d != want {
			t.Fatalf("action %d: got %+v, want %+v", i, d, want)
		}
	}

	mr.FastForward(20 * time.Second)
	d := mustAllow(t, l, "1.2.3.4")
	want := Decision{Limit: testLimit, RetryAfter: 40 * time.Second}
	if d != want {
		t.Fatalf("over limit: got %+v, want %+v", d, want)
	}
	if got := mr.TTL("rl:api:1.2.3.4"); got != 40*time.Second {
		t.Fatalf("counter TTL = %s, want 40s", got)
	}
}

func TestFixedWindow_ResetsAfterWindow(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for range testLimit + 1 {
		mustAllow(t, l, "1.2.3.4")
	}
	mr.FastForward(testWindow)

	d := mustAllow(t, l, "1.2.3.4")
	if !d.Allowed || d.Remaining != testLimit-1 {
		t.Fatalf("new window: got %+v, want allowed with %d remaining", d, testLimit-1)
	}
}

func TestFixedWindow_KeysAreIndependent(t *testing.T) {
	rdb, _ := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)

	for range testLimit + 1 {
		mustAllow(t, l, "1.2.3.4")
	}
	if d := mustAllow(t, l, "5.6.7.8"); !d.Allowed {
		t.Fatalf("other key must not be limited, got %+v", d)
	}
}

func TestFixedWindow_HealsCounterWithoutTTL(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)
	if err := mr.Set("rl:api:1.2.3.4", "100"); err != nil {
		t.Fatal(err)
	}

	d := mustAllow(t, l, "1.2.3.4")
	if d.Allowed || d.RetryAfter != testWindow {
		t.Fatalf("got %+v, want rejected with RetryAfter %s", d, testWindow)
	}
	mr.FastForward(testWindow)
	if d := mustAllow(t, l, "1.2.3.4"); !d.Allowed {
		t.Fatalf("after window: got %+v, want allowed", d)
	}
}

func TestFixedWindow_Concurrent(t *testing.T) {
	rdb, _ := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", 10, testWindow)

	if got := allowConcurrently(t, l, "1.2.3.4", 100); got != 10 {
		t.Fatalf("allowed %d of 100 concurrent actions, want 10", got)
	}
}

func TestFixedWindow_RedisError(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)
	mr.SetError("ERR boom")

	if _, err := l.Allow(context.Background(), "1.2.3.4"); err == nil {
		t.Fatal("want error")
	}
}

func TestFixedWindow_RejectionInLastMillisecond(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", testLimit, testWindow)
	if err := mr.Set("rl:api:1.2.3.4", "100"); err != nil {
		t.Fatal(err)
	}
	mr.SetTTL("rl:api:1.2.3.4", 500*time.Microsecond)

	d := mustAllow(t, l, "1.2.3.4")
	if d.Allowed || d.RetryAfter != time.Millisecond {
		t.Fatalf("got %+v, want rejected with RetryAfter 1ms", d)
	}
}

func TestFixedWindow_SubMillisecondWindow(t *testing.T) {
	rdb, mr := newTestRedis(t)
	l := NewRedis(rdb, "rl:api", 1, 500*time.Microsecond)

	if d := mustAllow(t, l, "1.2.3.4"); !d.Allowed {
		t.Fatalf("first: got %+v, want allowed", d)
	}
	if d := mustAllow(t, l, "1.2.3.4"); d.Allowed {
		t.Fatalf("second: got %+v, want rejected", d)
	}
	if got := mr.TTL("rl:api:1.2.3.4"); got != time.Millisecond {
		t.Fatalf("counter TTL = %s, want 1ms", got)
	}
}

func TestCeilMillis(t *testing.T) {
	for _, tc := range []struct{ in, want time.Duration }{
		{0, time.Millisecond},
		{500 * time.Microsecond, time.Millisecond},
		{time.Millisecond, time.Millisecond},
		{1500 * time.Microsecond, 2 * time.Millisecond},
		{time.Minute, time.Minute},
	} {
		if got := ceilMillis(tc.in); got != tc.want {
			t.Errorf("ceilMillis(%s) = %s, want %s", tc.in, got, tc.want)
		}
	}
}
