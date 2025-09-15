package tcp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
)

// StartTestServer starts a test HTTP server and returns its address and a cancel function
func StartTestServer() (string, context.CancelFunc) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := ts.Listener.Addr().String()
	return addr, ts.Close
}

// StartTestServerIPv6 starts a test HTTP server on IPv6 loopback and returns its address and a cancel function
func StartTestServerIPv6() (string, context.CancelFunc, error) {
	// Create a listener on IPv6 loopback address
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		return "", nil, err
	}

	// Create HTTP server with the IPv6 listener
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	// Start server in background
	go func() {
		_ = server.Serve(listener)
	}()

	addr := listener.Addr().String()
	cancel := func() {
		_ = server.Close()
	}

	return addr, cancel, nil
}
