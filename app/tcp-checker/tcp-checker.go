package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/tevino/tcp-shaker"
)

func parseFlag() (addr string, timeout time.Duration) {
	var timeoutMS int
	// Flag definition
	flag.IntVar(&timeoutMS, "timeout", 1000, "Timeout for the whole process(domain resolving is included)")
	flag.StringVar(&addr, "addr", "google.com:80", "TCP address to test")
	// Parse flags
	flag.Parse()
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		log.Fatalf("Can not resolve '%s': %s", addr, err)
	}
	timeout = time.Duration(timeoutMS) * time.Millisecond
	return
}

func main() {
	// Parse flag
	addr, timeout := parseFlag()
	log.Printf("Checking %s with timeout %s", addr, timeout)
	// Create checker
	s := tcp.NewChecker(true)
	// Init checker
	if err := s.InitChecker(); err != nil {
		log.Fatal("Initializing failed:", err)
	}
	// Check addr
	err := s.CheckAddr(addr, timeout)
	// Print error
	switch err {
	case tcp.ErrTimeout:
		log.Fatalf("Connect to '%s' timed out", addr)
	case nil:
		log.Printf("Connect to '%s' succeeded", addr)
	default:
		if e, ok := err.(*tcp.ErrConnect); ok {
			log.Fatalf("Connect to '%s' failed: %s", addr, e)
		} else {
			log.Fatalf("Error occurred while connecting to '%s': %s", addr, err)
		}
	}
}
