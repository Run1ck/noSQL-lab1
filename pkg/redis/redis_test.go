package redis

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	mr := miniredis.RunT(t)

	rdb, err := New(context.Background(), mr.Addr())
	require.NoError(t, err)
	_ = rdb.Close()
}

// Недоступный Redis — быстрая ошибка, а не секунды попыток подключения:
// на тех же настройках клиента работает fail-open лимита API.
func TestNew_UnreachableFailsFast(t *testing.T) {
	start := time.Now()
	_, err := New(context.Background(), "127.0.0.1:1")
	require.Error(t, err)
	require.Less(t, time.Since(start), 200*time.Millisecond, "want fast failure")
}

// Ответ опоздал дольше ReadTimeout — команда не повторяется: Redis её уже
// выполнил, и повтор засчитал бы действие дважды.
func TestNew_NoRetryAfterReadTimeout(t *testing.T) {
	mr := miniredis.RunT(t)
	var stall atomic.Bool
	addr := delayingProxy(t, mr.Addr(), &stall, timeout+200*time.Millisecond)

	rdb, err := New(context.Background(), addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	stall.Store(true)
	require.Error(t, rdb.Incr(context.Background(), "counter").Err(), "want read timeout")
	got, _ := mr.Get("counter")
	require.Equal(t, "1", got, "command was retried")
}

// delayingProxy проксирует TCP до addr. Если stall == true, следующий ответ
// сервера придерживается на delay, а stall сбрасывается — как короткое
// подвисание Redis: повтор по новому соединению прошёл бы без задержки.
func delayingProxy(t *testing.T, addr string, stall *atomic.Bool, delay time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			server, err := net.Dial("tcp", addr)
			if err != nil {
				_ = client.Close()
				return
			}
			go func() {
				_, _ = io.Copy(server, client)
				_ = server.Close()
			}()
			go func() {
				defer client.Close()
				buf := make([]byte, 4096)
				for {
					n, err := server.Read(buf)
					if err != nil {
						return
					}
					if stall.CompareAndSwap(true, false) {
						time.Sleep(delay)
					}
					if _, err := client.Write(buf[:n]); err != nil {
						return
					}
				}
			}()
		}
	}()
	return ln.Addr().String()
}
