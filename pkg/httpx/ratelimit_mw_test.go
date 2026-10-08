package httpx

import (
	"booking/pkg/ratelimit"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type fakeLimiter struct {
	d    ratelimit.Decision
	err  error
	keys []string
}

func (f *fakeLimiter) Allow(_ context.Context, key string) (ratelimit.Decision, error) {
	f.keys = append(f.keys, key)
	return f.d, f.err
}

// serve прогоняет запрос через RateLimit(l, ClientIP) и сообщает, дошёл ли он
// до обработчика.
func serve(l ratelimit.Limiter, path, remoteAddr string) (*httptest.ResponseRecorder, bool) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	RateLimit(l, ClientIP)(next).ServeHTTP(rec, req)
	return rec, reached
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Error.Code
}

func TestRateLimit_SkipsNonAPI(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Limit: 60}}

	for _, path := range []string{"/", "/index.html", "/catalog.js", "/api", "/apix"} {
		rec, reached := serve(l, path, "203.0.113.7:51234")
		if !reached || rec.Code != http.StatusOK {
			t.Errorf("%s: reached=%v code=%d, want passed through", path, reached, rec.Code)
		}
		if rec.Header().Get("X-RateLimit-Limit") != "" {
			t.Errorf("%s: unexpected rate limit headers", path)
		}
	}
	if len(l.keys) != 0 {
		t.Fatalf("limiter called for non-API paths: %v", l.keys)
	}
}

func TestRateLimit_Allowed(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Allowed: true, Limit: 60, Remaining: 59}}

	rec, reached := serve(l, "/api/services", "203.0.113.7:51234")
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("reached=%v code=%d, want passed through", reached, rec.Code)
	}
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "60" {
		t.Errorf("X-RateLimit-Limit = %q, want 60", got)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "59" {
		t.Errorf("X-RateLimit-Remaining = %q, want 59", got)
	}
	if len(l.keys) != 1 || l.keys[0] != "203.0.113.7" {
		t.Fatalf("limiter keys = %v, want [203.0.113.7]", l.keys)
	}
}

func TestRateLimit_Rejected(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Limit: 60, RetryAfter: 1500 * time.Millisecond}}

	rec, reached := serve(l, "/api/services", "203.0.113.7:51234")
	if reached {
		t.Fatal("rejected request must not reach the handler")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("code = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want 2 (rounded up)", got)
	}
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "60" {
		t.Errorf("X-RateLimit-Limit = %q, want 60", got)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, want 0", got)
	}
	if got := errorCode(t, rec); got != "rate_limited" {
		t.Errorf("error code = %q, want rate_limited", got)
	}
}

// captureLog перенаправляет slog в буфер до конца теста.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestRateLimit_FailOpen(t *testing.T) {
	logs := captureLog(t)
	reached := 0
	h := RateLimit(&fakeLimiter{err: errors.New("redis down")}, ClientIP)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached++ }))

	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Header().Get("X-RateLimit-Limit") != "" {
			t.Fatalf("code=%d limit=%q, want 200 without rate limit headers",
				rec.Code, rec.Header().Get("X-RateLimit-Limit"))
		}
	}
	if reached != 3 {
		t.Fatalf("reached handler %d times, want 3", reached)
	}
	// Отказ Redis пишется в лог один раз, а не на каждый запрос.
	if n := bytes.Count(logs.Bytes(), []byte("rate limit unavailable")); n != 1 {
		t.Fatalf("logged %d times, want 1:\n%s", n, logs)
	}
}

// Сквозной тест с настоящим лимитером на miniredis.
func TestRateLimit_WithRedisLimiter(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	l := ratelimit.NewRedis(rdb, "rl:api", 2, time.Minute)

	for i, want := range []struct {
		code      int
		remaining string
	}{
		{http.StatusOK, "1"},
		{http.StatusOK, "0"},
		{http.StatusTooManyRequests, "0"},
	} {
		rec, _ := serve(l, "/api/services", "203.0.113.7:51234")
		if rec.Code != want.code || rec.Header().Get("X-RateLimit-Remaining") != want.remaining {
			t.Fatalf("request %d: code=%d remaining=%q, want %d and %q", i+1,
				rec.Code, rec.Header().Get("X-RateLimit-Remaining"), want.code, want.remaining)
		}
	}
	if !mr.Exists("rl:api:203.0.113.7") {
		t.Fatal("counter key rl:api:203.0.113.7 not found")
	}
	if rec, _ := serve(l, "/api/services", "198.51.100.1:40000"); rec.Code != http.StatusOK {
		t.Fatalf("other IP: code=%d, want 200", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	for _, tc := range []struct {
		remoteAddr, forwarded, want string
	}{
		{"203.0.113.7:51234", "", "203.0.113.7"},
		{"[2001:db8::1]:443", "", "2001:db8::1"},
		{"203.0.113.7", "", "203.0.113.7"},
		{"203.0.113.7:51234", "1.1.1.1", "203.0.113.7"},
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		r.RemoteAddr = tc.remoteAddr
		if tc.forwarded != "" {
			r.Header.Set("X-Forwarded-For", tc.forwarded)
		}
		if got := ClientIP(r); got != tc.want {
			t.Errorf("ClientIP(%q, XFF=%q) = %q, want %q", tc.remoteAddr, tc.forwarded, got, tc.want)
		}
	}
}
