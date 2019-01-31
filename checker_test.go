package tcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

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
	err = c.CheckAddr("127.0.0.1:1", timeout)
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
	err = c.CheckAddr("10.0.0.0:1", timeout)
	assert(t, err == ErrTimeout)
}

func TestCheckAddrConcurrently(t *testing.T) {
	// Create checker
	c := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.CheckingLoop(ctx)

	var wg sync.WaitGroup

	check := func() {
		if err := c.CheckAddr("10.0.0.0:1", time.Millisecond*50); err == nil {
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
