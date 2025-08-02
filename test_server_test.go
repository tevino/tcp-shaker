
package tcp

import (
	"context"
	"net/http"
	"net/http/httptest"
)

// StartTestServer starts a test HTTP server and returns its address and a cancel function
func StartTestServer() (string, context.CancelFunc) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := ts.Listener.Addr().String()
	return addr, ts.Close
}
