package tcp

import (
	"context"
	"fmt"
	"net"

	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	AddrDead = "127.0.0.1:1"
)

var timeoutAddrs = []string{
	"10.255.255.1:80",
	"10.0.0.0:1",
}
var AddrTimeout = timeoutAddrs[0]

func _setAddrTimeout() {
	for _, addr := range timeoutAddrs {
		conn, err := net.DialTimeout("tcp", addr, time.Millisecond*50)
		if err == nil {
			_ = conn.Close()
			continue
		}
		if os.IsTimeout(err) {
			AddrTimeout = addr
			return
		}
	}
}

func init() {
	_setAddrTimeout()
}

// assert calls t.Fatal if the result is false
func assert(t *testing.T, result bool) {
	if !result {
		_, fileName, line, _ := runtime.Caller(1)
		t.Fatalf("Test failed: %s:%d", fileName, line)
	}
}

func ExampleChecker() {
	c := NewChecker()

	ctx, stopChecker := context.WithCancel(context.Background())
	defer stopChecker()
	go func() {
		if err := c.CheckingLoop(ctx); err != nil {
			fmt.Println("checking loop stopped due to fatal error: ", err)
		}
	}()

	<-c.WaitReady()

	timeout := time.Second * 1
	err := c.CheckAddr("example.com:80", timeout)
	switch err {
	case ErrTimeout:
		fmt.Println("Connect timed out")
	case nil:
		fmt.Println("Connect succeeded")
	default:
		fmt.Println("Error occurred while connecting: ", err)
	}
}

func TestCheckAddr(t *testing.T) {
	t.Parallel()
	var err error
	// Create checker
	c := NewChecker()

	// Start checker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = c.CheckingLoop(ctx)
	}()

	<-c.WaitReady()

	timeout := time.Second * 2
	// Check dead server
	err = c.CheckAddr(AddrDead, timeout)
	if runtime.GOOS == "linux" {
		_, ok := err.(*ErrConnect)
		assert(t, ok)
	} else {
		assert(t, err != nil)
	}
	// Launch a server for test
	addr, stop := StartTestServer()
	defer stop()
	// Check alive server
	err = c.CheckAddr(addr, timeout)
	assert(t, err == nil)
	// Check non-routable address, thus timeout
	err = c.CheckAddr(AddrTimeout, timeout)
	if err != ErrTimeout {
		t.Log("expected ErrTimeout, got ", err)
		t.FailNow()
	}
}

func TestCheckAddrConcurrently(t *testing.T) {
	// Create checker
	c := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = c.CheckingLoop(ctx)
	}()

	var wg sync.WaitGroup
	var failed bool

	check := func() {
		if err := c.CheckAddr(AddrTimeout, time.Millisecond*50); err == nil {
			failed = true
		}
		wg.Done()
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go check()
	}
	wg.Wait()

	if failed {
		t.Fatal("Concurrent testing failed")
	}
}
