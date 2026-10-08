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
	"github.com/stretchr/testify/require"
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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body.Error.Code
}

func TestRateLimit_SkipsNonAPI(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Limit: 60}}

	for _, path := range []string{"/", "/index.html", "/catalog.js", "/api", "/apix"} {
		rec, reached := serve(l, path, "203.0.113.7:51234")
		require.True(t, reached, path)
		require.Equal(t, http.StatusOK, rec.Code, path)
		require.Empty(t, rec.Header().Get("X-RateLimit-Limit"), path)
	}
	require.Empty(t, l.keys, "limiter called for non-API paths")
}

func TestRateLimit_Allowed(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Allowed: true, Limit: 60, Remaining: 59}}

	rec, reached := serve(l, "/api/services", "203.0.113.7:51234")
	require.True(t, reached)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "60", rec.Header().Get("X-RateLimit-Limit"))
	require.Equal(t, "59", rec.Header().Get("X-RateLimit-Remaining"))
	require.Equal(t, []string{"203.0.113.7"}, l.keys)
}

func TestRateLimit_Rejected(t *testing.T) {
	l := &fakeLimiter{d: ratelimit.Decision{Limit: 60, RetryAfter: 1500 * time.Millisecond}}

	rec, reached := serve(l, "/api/services", "203.0.113.7:51234")
	require.False(t, reached, "rejected request must not reach the handler")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "2", rec.Header().Get("Retry-After"), "rounded up")
	require.Equal(t, "60", rec.Header().Get("X-RateLimit-Limit"))
	require.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	require.Equal(t, "rate_limited", errorCode(t, rec))
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
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/services", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get("X-RateLimit-Limit"), "no rate limit headers")
	}
	require.Equal(t, 3, reached)
	// Отказ Redis пишется в лог один раз, а не на каждый запрос.
	require.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("rate limit unavailable")), logs.String())
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
		require.Equal(t, want.code, rec.Code, "request %d", i+1)
		require.Equal(t, want.remaining, rec.Header().Get("X-RateLimit-Remaining"), "request %d", i+1)
	}
	require.True(t, mr.Exists("rl:api:203.0.113.7"), "counter key not found")

	rec, _ := serve(l, "/api/services", "198.51.100.1:40000")
	require.Equal(t, http.StatusOK, rec.Code, "other IP")
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
		require.Equal(t, tc.want, ClientIP(r), "ClientIP(%q, XFF=%q)", tc.remoteAddr, tc.forwarded)
	}
}
