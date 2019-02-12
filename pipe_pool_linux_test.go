package tcp

import "testing"

func BenchmarkPipePoolDummyOK(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	addr, stop := _startTestServer()
	defer stop()
	_benchmarkChecker(b, c, addr)
}

func BenchmarkPipePoolSyncPoolOK(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolSyncPool()

	addr, stop := _startTestServer()
	defer stop()
	_benchmarkChecker(b, c, addr)
}

func BenchmarkPipePoolDummyErr(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	_benchmarkChecker(b, c, AddrDead)
}

func BenchmarkPipePoolSyncPoolErr(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolSyncPool()

	_benchmarkChecker(b, c, AddrDead)
}

func BenchmarkPipePoolDummyTimeout(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolDummy()

	_benchmarkChecker(b, c, AddrTimeout)
}

func BenchmarkPipePoolSyncPoolTimeout(b *testing.B) {
	c, cancel := _newChecker(b)
	defer cancel()
	c.pipePool = newPipePoolSyncPool()

	_benchmarkChecker(b, c, AddrTimeout)
}
