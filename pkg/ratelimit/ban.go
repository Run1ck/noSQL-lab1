package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// banScript учитывает действие пользователя и банит за превышение лимита.
// Проверка бана, инкремент и установка бана — один скрипт: иначе параллельные
// заявки проскочат между проверкой и записью.
//
//	KEYS[1] — счётчик действий за окно, rl:req:{key}
//	KEYS[2] — бан, ban:{key}
//	ARGV[1] — сколько действий разрешено за окно
//	ARGV[2] — длина окна в миллисекундах
//	ARGV[3] — длина бана в миллисекундах
//
// Возвращает {n, 0}, если действие пропущено (n — сколько действий в окне
// вместе с этим), и {-1, ttl} при отказе (ttl — сколько миллисекунд осталось
// до конца бана). Пока висит бан, действия не считаются. Действие сверх лимита
// уже отклоняется и ставит бан, а счётчик удаляется: после бана счёт с нуля,
// даже если окно длиннее бана.
var banScript = redis.NewScript(`
local ban = redis.call('PTTL', KEYS[2])
if ban > 0 then
	return {-1, ban}
end

local n = redis.call('INCR', KEYS[1])
if n == 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
if n > tonumber(ARGV[1]) then
	redis.call('SET', KEYS[2], '1', 'PX', ARGV[3])
	redis.call('DEL', KEYS[1])
	return {-1, tonumber(ARGV[3])}
end
return {n, 0}
`)

// Ban — больше limit действий за window ведут к бану на ttl; пока он висит,
// отклоняются все действия. Ключ — ID пользователя.
type Ban struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
	ttl    time.Duration
}

// NewBan возвращает бан с ключами rl:req:{key} и ban:{key}.
func NewBan(rdb *redis.Client, limit int, window, ttl time.Duration) *Ban {
	return &Ban{rdb: rdb, limit: limit, window: window, ttl: ttl}
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
