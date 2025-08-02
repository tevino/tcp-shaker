package tcp

import (
	"sync"
	"testing"
	"time"
)

func TestDefaultCheckerImmediatelyReady(t *testing.T) {
	checker := DefaultChecker()

	// Should be ready immediately after function returns
	select {
	case <-checker.WaitReady():
		// Expected behavior
	case <-time.After(100 * time.Millisecond):
		t.Error("Checker should be ready immediately after DefaultChecker() returns")
	}
}

func TestDefaultCheckerNotNil(t *testing.T) {
	if DefaultChecker() == nil {
		t.Fatalf("DefaultChecker() returned nil")
	}
}

func TestDefaultCheckerBasicFunctionality(t *testing.T) {
	// Start a test server to check against
	testAddr, stopServer := StartTestServer()
	defer stopServer()

	checker := DefaultChecker()

	// Test that we can perform a check
	err := checker.CheckAddr(testAddr, time.Second)
	if err != nil {
		t.Fatalf("Check against test server failed with error: %v", err)
	}
}

func TestDefaultCheckerSingleton(t *testing.T) {
	// Get first instance
	first := DefaultChecker()

	// Get second instance
	second := DefaultChecker()

	// They should be the same instance
	if first != second {
		t.Fatalf("DefaultChecker() did not return a singleton instance")
	}

	// Verify that the checker is ready
	select {
	case <-first.WaitReady():
		// Checker is ready as expected
	case <-time.After(5 * time.Second):
		t.Fatalf("DefaultChecker() did not become ready within 5 seconds")
	}
}

func TestDefaultCheckerConcurrentAccess(t *testing.T) {
	// Test concurrent access to the singleton
	ch := make(chan *Checker, 10)
	done := make(chan struct{})

	// Start multiple goroutines trying to get the checker
	for i := 0; i < 10; i++ {
		go func() {
			ch <- DefaultChecker()
			done <- struct{}{}
		}()
	}

	// Collect all results
	checkers := make([]*Checker, 10)
	for i := 0; i < 10; i++ {
		select {
		case checker := <-ch:
			checkers[i] = checker
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for checker")
		}
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for goroutine completion")
		}
	}

	// All checkers should be the same instance
	for _, checker := range checkers {
		if checker != checkers[0] {
			t.Errorf("Concurrent access returned different checker instances")
		}
	}
}

func TestDefaultCheckerOnceBehavior(t *testing.T) {
	// This test verifies that the once.Do function only executes once
	// even when called concurrently

	var wg sync.WaitGroup
	checkers := make([]*Checker, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			checkers[i] = DefaultChecker()
		}(i)
	}

	wg.Wait()

	// All should be the same instance
	for _, checker := range checkers {
		if checker != checkers[0] {
			t.Errorf("Different instances returned despite once.Do usage")
		}
	}
}
