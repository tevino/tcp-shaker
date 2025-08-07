package tcp

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func newChecker(b *testing.B) (*Checker, context.CancelFunc) {
	c := NewChecker()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = c.CheckingLoop(ctx)
	}()

	select {
	case <-time.After(time.Second):
		b.FailNow()
	case <-c.WaitReady():
	}
	return c, cancel
}

func benchmarkChecker(b *testing.B, c *Checker, addr string) {
	b.SetParallelism(runtime.NumCPU() * 10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// if timeout >= 1s, this func will run only once in the benchmark.
			_ = c.CheckAddr(addr, time.Millisecond*900)
		}
	})
	b.StopTimer()
}
