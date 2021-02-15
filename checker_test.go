package tcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	AddrDead      = "127.0.0.1:1"
	AddrIPV6Dead  = "[::1]:9001"
	AddrIPV6Alive = "[::1]:9002"
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
			conn.Close()
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
	err := c.CheckAddr("google.com:80", timeout)
	switch err {
	case ErrTimeout:
		fmt.Println("Connect to Google timed out")
	case nil:
		fmt.Println("Connect to Google succeeded")
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
	go c.CheckingLoop(ctx)

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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// Check alive server
	err = c.CheckAddr(ts.Listener.Addr().String(), timeout)
	assert(t, err == nil)
	ts.Close()
	// Check non-routable address, thus timeout
	err = c.CheckAddr(AddrTimeout, timeout)
	if err != ErrTimeout {
		t.Log("expected ErrTimeout, got ", err)
		t.FailNow()
	}
}

func TestCheckIPV6Addr(t *testing.T) {
	t.Parallel()
	var err error
	// Create checker
	c := NewChecker()

	// Start checker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.CheckingLoop(ctx)

	<-c.WaitReady()

	timeout := time.Second * 2

	// Launch a server for test
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	listener, _ := net.Listen("tcp6", AddrIPV6Alive)
	ts := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	ts.Start()

	// Check dead server
	err = c.CheckAddr(AddrIPV6Dead, timeout)
	if runtime.GOOS == "linux" {
		_, ok := err.(*ErrConnect)
		assert(t, ok)
	} else {
		assert(t, err != nil)
	}

	// Check alive server
	err = c.CheckAddr(AddrIPV6Alive, timeout)
	assert(t, err == nil)

	ts.Close()
}

func TestCheckAddrConcurrently(t *testing.T) {
	// Create checker
	c := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.CheckingLoop(ctx)

	var wg sync.WaitGroup

	check := func() {
		if err := c.CheckAddr(AddrTimeout, time.Millisecond*50); err == nil {
			t.Fatal("Concurrent testing failed")
		}
		wg.Done()
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go check()
	}
	wg.Wait()
}
