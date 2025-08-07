package tcp

import (
	"testing"

	"github.com/tevino/tcp-shaker/internal"
)

func BenchmarkPipePoolDummyOK(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	addr, stop := StartTestServer()
	defer stop()
	benchmarkChecker(b, c, addr)
}

func BenchmarkPipePoolSyncPoolOK(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = internal.NewPipePoolSyncPool()

	addr, stop := StartTestServer()
	defer stop()
	benchmarkChecker(b, c, addr)
}

func BenchmarkPipePoolDummyErr(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	benchmarkChecker(b, c, AddrDead)
}

func BenchmarkPipePoolSyncPoolErr(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = internal.NewPipePoolSyncPool()

	benchmarkChecker(b, c, AddrDead)
}

func BenchmarkPipePoolDummyTimeout(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	benchmarkChecker(b, c, AddrTimeout)
}

func BenchmarkPipePoolSyncPoolTimeout(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.pipePool = internal.NewPipePoolSyncPool()

	benchmarkChecker(b, c, AddrTimeout)
}
