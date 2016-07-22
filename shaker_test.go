package tcp

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime"
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

func ExampleShaker() {
	var s Shaker
	if err := s.InitShaker(); err != nil {
		log.Fatal("Shaker init failed:", err)
	}

	timeout := time.Second * 1
	err := s.TestAddr("google.com:80", timeout)
	switch err {
	case ErrTimeout:
		fmt.Println("Connect to Google timed out")
	case nil:
		fmt.Println("Connect to Google succeeded")
	default:
		if e, ok := err.(*ErrConnect); ok {
			fmt.Println("Connect to Google failed:", e)
		} else {
			fmt.Println("Error occurred while connecting:", err)
		}
	}
}

func TestTestAddr(t *testing.T) {
	var err error
	// Create shaker
	s := Shaker{}
	if err := s.InitShaker(); err != nil {
		t.Fatal("Shaker init failed:", err)
	}
	timeout := time.Second * 2
	// Test dead server
	err = s.TestAddr("127.0.0.1:1", timeout)
	_, ok := err.(*ErrConnect)
	assert(t, ok)
	// Launch a server for test
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// Test alive server
	err = s.TestAddr(ts.Listener.Addr().String(), timeout)
	assert(t, err == nil)
	ts.Close()
	// Test non-routable address, thus timeout
	err = s.TestAddr("10.0.0.0:1", timeout)
	assert(t, err == ErrTimeout)
}

func TestClose(t *testing.T) {
	var err error
	// Create shaker
	s := Shaker{}
	assert(t, !s.Ready())
	if err := s.InitShaker(); err != nil {
		t.Fatal("Shaker init failed:", err)
	}
	assert(t, s.Ready())
	// Close the shaker
	s.Close()
	assert(t, !s.Ready())
	// Init the shaker again
	s.InitShaker()
	assert(t, s.Ready())
	timeout := time.Second * 2
	// Test dead server
	err = s.TestAddr("127.0.0.1:1", timeout)
	_, ok := err.(*ErrConnect)
	assert(t, ok)
	// Launch a server for test
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// Test alive server
	err = s.TestAddr(ts.Listener.Addr().String(), timeout)
	assert(t, err == nil)
	ts.Close()
	// Test non-routable address, thus timeout
	err = s.TestAddr("10.0.0.0:1", timeout)
	assert(t, err == ErrTimeout)
}
