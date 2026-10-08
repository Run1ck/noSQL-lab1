package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var windowScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl == -1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end
return {n, ttl}
`)

type FixedWindow struct {
	rdb    *redis.Client
	prefix string
	limit  int
	window time.Duration
}

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
		d.RetryAfter = max(ttl, time.Millisecond)
	}
	return d, nil
}

func ceilMillis(d time.Duration) time.Duration {
	return max(time.Millisecond, (d + time.Millisecond - 1).Truncate(time.Millisecond))
}
