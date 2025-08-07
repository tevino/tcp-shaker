package tcp

import (
	"testing"

	"github.com/tevino/tcp-shaker/internal"
)

func BenchmarkResultPipesMUOK(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	addr, stop := StartTestServer()
	defer stop()
	benchmarkChecker(b, c, addr)
}

func BenchmarkResultPipesSyncMapOK(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = internal.NewResultPipesSyncMap()

	addr, stop := StartTestServer()
	defer stop()
	benchmarkChecker(b, c, addr)
}
func BenchmarkResultPipesMUErr(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	benchmarkChecker(b, c, AddrDead)
}

func BenchmarkResultPipesSyncMapErr(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = internal.NewResultPipesSyncMap()

	benchmarkChecker(b, c, AddrDead)
}

func BenchmarkResultPipesMUTimeout(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = newResultPipesMU()

	benchmarkChecker(b, c, AddrTimeout)
}

func BenchmarkResultPipesSyncMapTimeout(b *testing.B) {
	c, cancel := newChecker(b)
	defer cancel()
	c.resultPipes = internal.NewResultPipesSyncMap()

	benchmarkChecker(b, c, AddrTimeout)
}
