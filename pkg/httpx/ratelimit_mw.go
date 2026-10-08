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

// warnEvery — как часто писать в лог, что лимит недоступен: при отказе Redis
// под нагрузкой строка на каждый запрос — это тысячи строк в секунду.
const warnEvery = 10 * time.Second

// RateLimit ограничивает частоту запросов к /api/*; ключ лимита — key(r), в main
// это ClientIP. Остальные пути (статика UI) не считаются: main оборачивает весь
// mux, поэтому фильтр по пути — здесь.
//
// Ответ несёт X-RateLimit-Limit и X-RateLimit-Remaining; сверх лимита — 429
// rate_limited с Retry-After. Если хранилище лимита недоступно, запрос
// пропускается (fail-open), а ошибка пишется в лог не чаще раза в warnEvery.
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
				warn.log(err)
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

// throttledWarn пишет в лог не чаще раза в warnEvery и сообщает, сколько
// записей пропущено с прошлого раза.
type throttledWarn struct {
	last       atomic.Int64 // UnixNano последней записи
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

// ClientIP — IP клиента из адреса TCP-соединения. X-Forwarded-For и похожие
// заголовки не читаем: их присылает сам клиент, и подменой обходится лимит.
// За обратным прокси все клиенты получили бы IP прокси — у нас его нет.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
