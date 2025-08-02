package tcp

import (
	"context"
	"testing"
	"time"
)

func TestCheckerReadyOK(t *testing.T) {
	t.Parallel()
	c := NewChecker()
	assert(t, !c.IsReady())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.CheckingLoop(ctx)
	select {
	case <-time.After(time.Second):
		t.FailNow()
	case <-c.WaitReady():
	}
}

func TestStopNStartChecker(t *testing.T) {
	t.Parallel()

	// Create checker
	c := NewChecker()

	// Start checker
	ctx, cancel := context.WithCancel(context.Background())
	loopStopped := make(chan bool)
	go func() {
		err := c.CheckingLoop(ctx)
		assert(t, err == nil)
		loopStopped <- true
	}()

	// Close the checker
	cancel()
	<-loopStopped

	// Start the checker again
	ctx, cancel = context.WithCancel(context.Background())
	defer func() {
		cancel()
		<-loopStopped
	}()
	go func() {
		err := c.CheckingLoop(ctx)
		assert(t, err == nil)
		loopStopped <- true
	}()

	// Ensure the check works
	_testChecker(t, c)
}

func _testChecker(t *testing.T, c *Checker) {
	select {
	case <-c.WaitReady():
	case <-time.After(time.Second):
	}

	timeout := time.Second * 2
	// Check dead server
	err := c.CheckAddr(AddrDead, timeout)
	_, ok := err.(*ErrConnect)
	assert(t, ok)

	// Launch a server for test
	addr, stop := StartTestServer()
	defer stop()

	// Check alive server
	err = c.CheckAddr(addr, timeout)
	assert(t, err == nil)

	// Check non-routable address, thus timeout
	err = c.CheckAddr(AddrTimeout, timeout)
	assert(t, err == ErrTimeout)
}
