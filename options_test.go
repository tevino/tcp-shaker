package tcp

import (
	"context"
	"testing"
	"time"
)

// setupTestChecker creates a checker with CheckingLoop running and waits until ready
func setupTestChecker(t *testing.T) (*Checker, context.CancelFunc) {
	checker := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = checker.CheckingLoop(ctx)
	}()
	<-checker.WaitReady()
	return checker, cancel
}

// testWithChecker runs a test function with a prepared checker and test server
func testWithChecker(t *testing.T, testFunc func(*testing.T, *Checker, string)) {
	checker, cancel := setupTestChecker(t)
	defer cancel()

	testAddr, stopServer := StartTestServer()
	defer stopServer()

	testFunc(t, checker, testAddr)
}

func TestOptions(t *testing.T) {
	// Test default options
	opts := DefaultOptions()
	if opts.Network != "tcp" {
		t.Errorf("expected default network to be 'tcp', got %s", opts.Network)
	}
	if opts.Timeout != 3*time.Second {
		t.Errorf("expected default timeout to be 3s, got %v", opts.Timeout)
	}
	if !opts.ZeroLinger {
		t.Error("expected default ZeroLinger to be true")
	}

	// Test fluent interface
	customOpts := DefaultOptions().
		WithTimeout(5 * time.Second).
		WithNetwork("tcp6").
		WithZeroLinger(false)

	if customOpts.Network != "tcp6" {
		t.Errorf("expected network to be 'tcp6', got %s", customOpts.Network)
	}
	if customOpts.Timeout != 5*time.Second {
		t.Errorf("expected timeout to be 5s, got %v", customOpts.Timeout)
	}
	if customOpts.ZeroLinger {
		t.Error("expected ZeroLinger to be false")
	}
}
