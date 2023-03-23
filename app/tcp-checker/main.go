package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

// Config contains all available options.
type Config struct {
	Addr        string
	Timeout     time.Duration
	Requests    int
	Concurrency int
	Verbose     bool
}

func parseConfig() *Config {
	var conf Config
	var timeoutMS int
	// Flag definition
	flag.IntVar(&timeoutMS, "t", 1000, "Timeout in millisecond for the whole checking process(domain resolving is included)")
	flag.StringVar(&conf.Addr, "a", "google.com:80", "TCP address to test")
	flag.IntVar(&conf.Requests, "n", 1, "Number of requests to perform")
	flag.IntVar(&conf.Concurrency, "c", 1, "Number of checks to perform simultaneously")
	flag.BoolVar(&conf.Verbose, "v", false, "Print more logs e.g. error detail")
	// Parse flags
	flag.Parse()
	if _, err := net.ResolveTCPAddr("tcp", conf.Addr); err != nil {
		log.Fatalf("Can not resolve '%s': %s", conf.Addr, err)
	}
	conf.Timeout = time.Duration(timeoutMS) * time.Millisecond
	return &conf
}

func setupSignal(exit chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(exit)
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	log.SetFlags(0)
	conf := parseConfig()

	log.Printf(`Checking %s with the following configurations:
    Timeout: %s
   Requests: %d
Concurrency: %d`, conf.Addr, conf.Timeout, conf.Requests, conf.Concurrency)

	checker := NewConcurrentChecker(conf)
	defer checker.Stop()

	var exit = make(chan bool)
	go setupSignal(exit)
	startedAt := time.Now()

	var ctx, cancel = context.WithCancel(context.Background())
	if err := checker.Launch(ctx); err != nil {
		log.Fatal("Initializing failed: ", err)
	}
	select {
	case <-exit:
	case <-checker.Wait():
	}

	duration := time.Now().Sub(startedAt)
	if conf.Verbose {
		log.Println("Canceling checking loop")
	}
	cancel()

	log.Println("")
	log.Printf("Finished %d/%d checks in %s\n", checker.Count(CRequest), conf.Requests, duration)
	log.Printf("  Succeed: %d\n", checker.Count(CSucceed))
	log.Printf("  Errors: connect %d, timeout %d, other %d\n", checker.Count(CErrConnect), checker.Count(CErrTimeout), checker.Count(CErrOther))
}
