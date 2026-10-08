package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var banScript = redis.NewScript(`
local ban = redis.call('PTTL', KEYS[2])
if ban == -1 then
	redis.call('PEXPIRE', KEYS[2], ARGV[3])
	ban = tonumber(ARGV[3])
end
if ban > 0 then
	return {-1, ban}
end

local n = redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) == -1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
if n > tonumber(ARGV[1]) then
	redis.call('SET', KEYS[2], '1', 'PX', ARGV[3])
	redis.call('DEL', KEYS[1])
	return {-1, tonumber(ARGV[3])}
end
return {n, 0}
`)

type Ban struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
	ttl    time.Duration
}

func NewBan(rdb *redis.Client, limit int, window, ttl time.Duration) *Ban {
	return &Ban{rdb: rdb, limit: limit, window: ceilMillis(window), ttl: ceilMillis(ttl)}
}

func (b *Ban) Allow(ctx context.Context, key string) (Decision, error) {
	res, err := banScript.Run(ctx, b.rdb,
		[]string{"rl:req:" + key, "ban:" + key},
		b.limit, b.window.Milliseconds(), b.ttl.Milliseconds()).Int64Slice()
	if err != nil {
		return Decision{}, fmt.Errorf("ban limit: %w", err)
	}
	n, banLeft := int(res[0]), time.Duration(res[1])*time.Millisecond

	if n < 0 {
		return Decision{Limit: b.limit, RetryAfter: banLeft}, nil
	}
	return Decision{Allowed: true, Limit: b.limit, Remaining: b.limit - n}, nil
}
