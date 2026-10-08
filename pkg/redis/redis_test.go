package redis

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestNew(t *testing.T) {
	mr := miniredis.RunT(t)

	rdb, err := New(context.Background(), mr.Addr())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_ = rdb.Close()
}

func TestNew_UnreachableFailsFast(t *testing.T) {
	start := time.Now()
	if _, err := New(context.Background(), "127.0.0.1:1"); err == nil {
		t.Fatal("want error for unreachable Redis")
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("New took %s, want fast failure", d)
	}
}

func TestNew_SilentServerTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()

	start := time.Now()
	if _, err := New(context.Background(), ln.Addr().String()); err == nil {
		t.Fatal("want timeout error")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("New took %s, want about ReadTimeout (%s)", d, timeout)
	}
}

func TestNew_NoRetryAfterReadTimeout(t *testing.T) {
	mr := miniredis.RunT(t)
	var stall atomic.Bool
	addr := delayingProxy(t, mr.Addr(), &stall, timeout+200*time.Millisecond)

	rdb, err := New(context.Background(), addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	stall.Store(true)
	if err := rdb.Incr(context.Background(), "counter").Err(); err == nil {
		t.Fatal("want read timeout")
	}
	if got, _ := mr.Get("counter"); got != "1" {
		t.Fatalf("counter = %q, want 1: command was retried", got)
	}
}

func delayingProxy(t *testing.T, addr string, stall *atomic.Bool, delay time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
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
