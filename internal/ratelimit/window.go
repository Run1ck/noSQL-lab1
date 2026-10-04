package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// windowScript учитывает действие в фиксированном окне.
//
//	KEYS[1] — счётчик окна, prefix:{key}
//	ARGV[1] — длина окна в миллисекундах
//
// Возвращает {n, ttl}: n — сколько действий в окне вместе с этим, ttl — сколько
// миллисекунд осталось до конца окна. TTL ставится, если его нет: на первом
// действии окна и у счётчика, который почему-то остался без TTL, — иначе такой
// ключ блокировал бы навсегда.
var windowScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl == -1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end
return {n, ttl}
`)

// FixedWindow — не больше limit действий по ключу за window. Счётчик живёт одно
// окно и пропадает вместе с ключом. На стыке двух окон можно успеть до 2×limit
// действий — плата за то, что всё укладывается в один INCR.
type FixedWindow struct {
	rdb    *redis.Client
	prefix string
	limit  int
	window time.Duration
}

// NewRedis возвращает лимит с ключами prefix:{key}, например rl:api:{ip}.
func NewRedis(rdb *redis.Client, prefix string, limit int, window time.Duration) *FixedWindow {
	return &FixedWindow{rdb: rdb, prefix: prefix, limit: limit, window: ceilMillis(window)}
}

func (l *FixedWindow) Allow(ctx context.Context, key string) (Decision, error) {
	res, err := windowScript.Run(ctx, l.rdb,
		[]string{l.prefix + ":" + key}, l.window.Milliseconds()).Int64Slice()
	if err != nil {
		return Decision{}, fmt.Errorf("rate limit: %w", err)
	}
	n, ttl := int(res[0]), time.Duration(res[1])*time.Millisecond

	d := Decision{
		Allowed:   n <= l.limit,
		Limit:     l.limit,
		Remaining: max(0, l.limit-n),
	}
	if !d.Allowed {
		// В последнюю миллисекунду окна PTTL отдаёт 0, а отказ обязан
		// нести RetryAfter > 0.
		d.RetryAfter = max(ttl, time.Millisecond)
	}
	return d, nil
}

// ceilMillis округляет d вверх до целых миллисекунд, минимум 1 мс. TTL в Redis
// задаются в миллисекундах, а Milliseconds() обрезает: окно 500µs стало бы 0,
// PEXPIRE 0 удаляет ключ (лимит выключен), а SET … PX 0 — ошибка.
func ceilMillis(d time.Duration) time.Duration {
	return max(time.Millisecond, (d + time.Millisecond - 1).Truncate(time.Millisecond))
}
