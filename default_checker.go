package tcp

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	defaultChecker *Checker
	once           sync.Once
)

// DefaultChecker returns a shared singleton instance of the Checker.
//
// It starts the Checker's CheckingLoop in a goroutine with a context that
// listens for system signals (SIGINT, SIGTERM) to stop gracefully. This function
// blocks until the Checker is ready for use.
func DefaultChecker() *Checker {
	once.Do(func() {
		defaultChecker = NewChecker()

		go func() {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// If the checking loop stops with an error, we log it.
			// We use the standard log package to avoid external dependencies.
			if err := defaultChecker.CheckingLoop(ctx); err != nil {
				log.Printf("tcpshaker: TCP checking loop stopped with an error: %v", err)
			}
		}()

		// Wait for the checker to be ready to ensure initialization is complete
		// before returning the instance.
		<-defaultChecker.WaitReady()
	})

	return defaultChecker
}
