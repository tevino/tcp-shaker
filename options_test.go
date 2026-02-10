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
	if opts.Mark != 0 {
		t.Errorf("expected default Mark to be 0, got %d", opts.Mark)
	}

	// Test fluent interface
	customOpts := DefaultOptions().
		WithTimeout(5 * time.Second).
		WithNetwork("tcp6").
		WithZeroLinger(false).
		WithMark(100)

	if customOpts.Network != "tcp6" {
		t.Errorf("expected network to be 'tcp6', got %s", customOpts.Network)
	}
	if customOpts.Timeout != 5*time.Second {
		t.Errorf("expected timeout to be 5s, got %v", customOpts.Timeout)
	}
	if customOpts.ZeroLinger {
		t.Error("expected ZeroLinger to be false")
	}
	if customOpts.Mark != 100 {
		t.Errorf("expected Mark to be 100, got %d", customOpts.Mark)
	}
}

func TestCheckerWithOptions(t *testing.T) {
	testWithChecker(t, func(t *testing.T, checker *Checker, testAddr string) {
		// Test basic functionality with default options
		opts := DefaultOptions().WithTimeout(2 * time.Second)
		err := checker.CheckAddrWithOptions(testAddr, opts)
		if err != nil {
			t.Errorf("Connection to test server failed: %v", err)
		}

		// Test IPv4 specific
		opts4 := DefaultOptions().WithTimeout(2 * time.Second).WithNetwork("tcp4")
		err = checker.CheckAddrWithOptions(testAddr, opts4)
		if err != nil {
			t.Errorf("IPv4 connection to test server failed: %v", err)
		}
	})
}

func TestIPv6Support(t *testing.T) {
	checker, cancel := setupTestChecker(t)
	defer cancel()

	// Try to start IPv6 server
	testAddr6, stopServer6, err := StartTestServerIPv6()
	if err != nil {
		t.Skipf("Skipping IPv6 test: %v", err)
		return
	}
	defer stopServer6()

	// Test IPv6 connection with tcp6 network
	opts6 := DefaultOptions().WithTimeout(2 * time.Second).WithNetwork("tcp6")
	err = checker.CheckAddrWithOptions(testAddr6, opts6)
	if err != nil {
		t.Errorf("IPv6 connection to test server failed: %v", err)
	}

	// Test IPv6 connection with tcp network (should also work)
	optsGeneral := DefaultOptions().WithTimeout(2 * time.Second).WithNetwork("tcp")
	err = checker.CheckAddrWithOptions(testAddr6, optsGeneral)
	if err != nil {
		t.Errorf("General TCP connection to IPv6 test server failed: %v", err)
	}

	// Test that tcp4 fails on IPv6 address (should fail)
	opts4 := DefaultOptions().WithTimeout(1 * time.Second).WithNetwork("tcp4")
	err = checker.CheckAddrWithOptions(testAddr6, opts4)
	if err == nil {
		t.Error("Expected tcp4 connection to IPv6 address to fail, but it succeeded")
	} else {
		t.Logf("tcp4 connection to IPv6 address failed as expected: %v", err)
	}
}

func TestMarkOption(t *testing.T) {
	testWithChecker(t, func(t *testing.T, checker *Checker, testAddr string) {
		// Test with mark set (only meaningful on Linux)
		opts := DefaultOptions().WithTimeout(2 * time.Second).WithMark(42)
		err := checker.CheckAddrWithOptions(testAddr, opts)
		if err != nil {
			if err.Error() == "operation not permitted" {
				t.Skipf("Skipping mark test: need root or CAP_NET_ADMIN to set SO_MARK (%v)", err)
			}
			t.Errorf("Connection with mark to test server failed: %v", err)
		}
	})
}

func TestBackwardCompatibility(t *testing.T) {
	testWithChecker(t, func(t *testing.T, checker *Checker, testAddr string) {
		// Old API should still work
		err := checker.CheckAddr(testAddr, 2*time.Second)
		if err != nil {
			t.Errorf("Old API connection to test server failed: %v", err)
		}

		// CheckAddrZeroLinger should use new implementation internally
		err = checker.CheckAddrZeroLinger(testAddr, 2*time.Second, true)
		if err != nil {
			t.Errorf("CheckAddrZeroLinger connection to test server failed: %v", err)
		}
	})
}
