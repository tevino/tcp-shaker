package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"
)

// Run the program in CLI / one-off mode.
func cliMode(cliArgs *CLIConfig) {
	log.Printf(`Checking %s with the following configurations:
    Timeout: %s
   Requests: %d
Concurrency: %d`, cliArgs.Addr, cliArgs.Timeout, cliArgs.Requests, cliArgs.Concurrency)

	checker := NewConcurrentChecker(cliArgs, cliArgs.Addr)
	defer checker.Stop()

	var exit = make(chan bool)
	go gracefulCLIShutdown(exit)
	startedAt := time.Now()

	var ctx, cancel = context.WithCancel(context.Background())
	if err := checker.Launch(ctx); err != nil {
		log.Fatal("Initializing failed: ", err)
	}
	select {
	case <-exit:
	case <-checker.Wait():
	}

	duration := time.Since(startedAt)
	if cliArgs.Verbose {
		log.Println("Canceling checking loop")
	}
	cancel()

	log.Println("")
	log.Printf("Finished %d/%d checks in %s\n", checker.Count(CRequest), cliArgs.Requests, duration)
	log.Printf("  Succeed: %d\n", checker.Count(CSucceed))
	log.Printf("  Errors: connect %d, timeout %d, other %d\n", checker.Count(CErrConnect), checker.Count(CErrTimeout), checker.Count(CErrOther))
}

func gracefulCLIShutdown(exit chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(exit)
}
