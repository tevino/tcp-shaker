package tcp

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func _newChecker(b *testing.B) (*Checker, context.CancelFunc) {
	c := NewChecker()

	ctx, cancel := context.WithCancel(context.Background())
	go c.CheckingLoop(ctx)

	select {
	case <-time.After(time.Second):
		b.FailNow()
	case <-c.WaitReady():
	}
	return c, cancel
}

func _benchmarkChecker(b *testing.B, c *Checker, addr string) {
	b.SetParallelism(runtime.NumCPU() * 10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.CheckAddr(addr, time.Second)
		}
	})
	b.StopTimer()
}

func BenchmarkResultPipesMUOK(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	addr, stop := _startTestServer()
	defer stop()
	_benchmarkChecker(b, c, addr)
}

func BenchmarkResultPipesSyncMapOK(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesSyncMap()

	addr, stop := _startTestServer()
	defer stop()
	_benchmarkChecker(b, c, addr)
}
func BenchmarkResultPipesMUErr(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	_benchmarkChecker(b, c, AddrDead)
}

func BenchmarkResultPipesSyncMapErr(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesSyncMap()

	_benchmarkChecker(b, c, AddrDead)
}

func BenchmarkResultPipesMUTimeout(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	_benchmarkChecker(b, c, AddrTimeout)
}

func BenchmarkResultPipesSyncMapTimeout(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesSyncMap()

	_benchmarkChecker(b, c, AddrTimeout)
}
