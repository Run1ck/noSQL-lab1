package httpx

import (
	"booking/pkg/ratelimit"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const warnEvery = 10 * time.Second

func RateLimit(l ratelimit.Limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	var warn throttledWarn
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			d, err := l.Allow(r.Context(), key(r))
			if err != nil {
				if r.Context().Err() == nil {
					warn.log(err)
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(d.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(d.Remaining))
			if !d.Allowed {
				WriteError(w, &ratelimit.LimitedError{RetryAfter: d.RetryAfter})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type throttledWarn struct {
	last       atomic.Int64
	suppressed atomic.Int64
}

func (t *throttledWarn) log(err error) {
	now := time.Now().UnixNano()
	last := t.last.Load()
	if now-last < int64(warnEvery) || !t.last.CompareAndSwap(last, now) {
		t.suppressed.Add(1)
		return
	}
	slog.Warn("rate limit unavailable, requests allowed",
		"err", err, "suppressed", t.suppressed.Swap(0))
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
