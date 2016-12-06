package tcp

import (
	"fmt"
	"log"
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
	s := NewChecker(true)
	if err := s.InitChecker(); err != nil {
		log.Fatal("Checker init failed:", err)
	}

	timeout := time.Second * 1
	err := s.CheckAddr("google.com:80", timeout)
	switch err {
	case ErrTimeout:
		fmt.Println("Connect to Google timed out")
	case nil:
		fmt.Println("Connect to Google succeeded")
	default:
		fmt.Println("Error occurred while connecting:", err)
	}
}

func TestCheckAddr(t *testing.T) {
	var err error
	// Create checker
	s := NewChecker(true)
	if err := s.InitChecker(); err != nil {
		t.Fatal("Checker init failed:", err)
	}
	timeout := time.Second * 2
	// Check dead server
	err = s.CheckAddr("127.0.0.1:1", timeout)
	_, ok := err.(*ErrConnect)
	assert(t, ok)
	// Launch a server for test
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// Check alive server
	err = s.CheckAddr(ts.Listener.Addr().String(), timeout)
	assert(t, err == nil)
	ts.Close()
	// Check non-routable address, thus timeout
	err = s.CheckAddr("10.0.0.0:1", timeout)
	assert(t, err == ErrTimeout)
}

func TestClose(t *testing.T) {
	var err error
	// Create checker
	s := NewChecker(true)
	assert(t, !s.Ready())
	if err := s.InitChecker(); err != nil {
		t.Fatal("Checker init failed:", err)
	}
	assert(t, s.Ready())
	// Close the checker
	err = s.Close()
	assert(t, err == nil)
	assert(t, !s.Ready())
	// Init the checker again
	err = s.InitChecker()
	assert(t, err == nil)
	assert(t, s.Ready())
	timeout := time.Second * 2
	// Check dead server
	err = s.CheckAddr("127.0.0.1:1", timeout)
	_, ok := err.(*ErrConnect)
	assert(t, ok)
	// Launch a server for test
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// Check alive server
	err = s.CheckAddr(ts.Listener.Addr().String(), timeout)
	assert(t, err == nil)
	ts.Close()
	// Check non-routable address, thus timeout
	err = s.CheckAddr("10.0.0.0:1", timeout)
	assert(t, err == ErrTimeout)
}

func TestCheckAddrConcurrently(t *testing.T) {
	// Create checker
	s := NewChecker(true)
	if err := s.InitChecker(); err != nil {
		t.Fatal("Checker init failed:", err)
	}

	var wg sync.WaitGroup

	tasks := make(chan bool, 10)
	worker := func() {
		for range tasks {
			if err := s.CheckAddr("10.0.0.0:1", time.Second); err == nil {
				t.Fatal("Concurrent testing failed")
			}
			wg.Done()
		}
	}

	for i := 0; i < 10; i++ {
		go worker()
	}

	for i := 0; i < 1000; i++ {
		tasks <- true
		wg.Add(1)
	}
	wg.Wait()
	close(tasks)
}
