// Package ratelimit — атомарное ограничение частоты действий. Один и тот же
// Limiter работает в двух ролях:
//   - лимит REST API: ключ — IP клиента, не больше RATE_LIMIT запросов
//     за RATE_WINDOW;
//   - бан за частые заявки: ключ — ID пользователя; больше BAN_LIMIT заявок
//     за BAN_WINDOW — бан на BAN_TTL, до его конца все попытки отклоняются.
package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// Decision — решение по одному действию.
type Decision struct {
	Allowed bool
	// Limit — сколько действий разрешено за окно.
	Limit int
	// Remaining — сколько ещё осталось в текущем окне, не меньше 0.
	Remaining int
	// RetryAfter — через сколько можно повторить; больше 0 только при отказе.
	RetryAfter time.Duration
}

type Limiter interface {
	// Allow учитывает одно действие по ключу и решает, пропустить ли его.
	// Проверка и инкремент — одна атомарная операция в Redis (Lua-скрипт),
	// иначе параллельные запросы пробивают лимит.
	//
	// Ошибка означает только недоступность хранилища: вызывающий тогда
	// пропускает действие (fail-open) и пишет ошибку в лог.
	Allow(ctx context.Context, key string) (Decision, error)
}

// LimitedError — отказ лимитера. httpx.WriteError отвечает на неё 429
// с заголовком Retry-After и кодом rate_limited или banned.
type LimitedError struct {
	Banned     bool
	RetryAfter time.Duration
}

func (e *LimitedError) Error() string {
	if e.Banned {
		return fmt.Sprintf("too many requests, banned for %s", e.RetryAfter.Round(time.Second))
	}
	return fmt.Sprintf("rate limit exceeded, retry after %s", e.RetryAfter.Round(time.Second))
}
