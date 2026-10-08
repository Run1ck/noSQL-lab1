package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, mr
}

func mustAllow(t *testing.T, l Limiter, key string) Decision {
	t.Helper()
	d, err := l.Allow(context.Background(), key)
	require.NoError(t, err, "Allow(%q)", key)
	return d
}

// allowConcurrently делает n параллельных Allow по одному ключу и возвращает,
// сколько из них пропущено.
func allowConcurrently(t *testing.T, l Limiter, key string, n int) int {
	t.Helper()
	var (
		wg      sync.WaitGroup
		allowed atomic.Int64
	)
	for range n {
		wg.Go(func() {
			d, err := l.Allow(context.Background(), key)
			if err != nil {
				// Не require: FailNow нельзя звать не из горутины теста.
				t.Errorf("Allow: %v", err)
				return
			}
			if d.Allowed {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()
	return int(allowed.Load())
}
