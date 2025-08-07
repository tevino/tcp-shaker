package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	tcpshaker "github.com/tevino/tcp-shaker"
)

// Counter is an atomic counter for multiple metrics.
type Counter struct {
	counters map[int]*uint64
}

// NewCounter creates Counter with given IDs.
func NewCounter(ids ...int) *Counter {
	counter := &Counter{}
	counter.Declare(ids...)
	return counter
}

// Declare declares the ID of counters.
// NOTE: This must be called before counting, and should only be called once.
func (c *Counter) Declare(ids ...int) {
	c.counters = make(map[int]*uint64, len(ids))
	for _, id := range ids {
		var i uint64
		c.counters[id] = &i
	}
}

// Inc increases the counter of given ID by one and returns the new value.
func (c *Counter) Inc(i int) uint64 {
	return atomic.AddUint64(c.counters[i], 1)
}

// Count returns the value of counter with given ID.
func (c *Counter) Count(i int) uint64 {
	return atomic.LoadUint64(c.counters[i])
}

// Available counter names.
const (
	CRequest int = iota
	CSucceed
	CErrConnect
	CErrTimeout
	CErrOther
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
	flag.StringVar(&conf.Addr, "a", "example.com:80", "TCP address to test")
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

// ConcurrentChecker is a wrapper of tcpshaker.Checker with concurrent checking capabilities.
type ConcurrentChecker struct {
	conf    *Config
	counter *Counter
	checker *tcpshaker.Checker
	queue   chan bool
	closed  chan bool
	wg      sync.WaitGroup
}

// NewConcurrentChecker creates a checker.
func NewConcurrentChecker(conf *Config) *ConcurrentChecker {
	return &ConcurrentChecker{
		conf:    conf,
		counter: NewCounter(CRequest, CSucceed, CErrConnect, CErrTimeout, CErrOther),
		checker: tcpshaker.NewChecker(),
		queue:   make(chan bool),
		closed:  make(chan bool),
	}
}

// Count returns the count of given ID.
func (cc *ConcurrentChecker) Count(i int) uint64 {
	return cc.counter.Count(i)
}

// Launch initialize the checker.
func (cc *ConcurrentChecker) Launch(ctx context.Context) {
	go func() {
		err := cc.checker.CheckingLoop(ctx)
		if err != nil {
			log.Fatal("Error during checking loop: ", err)
		}
	}()

	for i := 0; i < cc.conf.Concurrency; i++ {
		go cc.worker()
	}
	cc.wg.Add(cc.conf.Requests)

	if cc.conf.Verbose {
		log.Println("Waiting for checker to be ready")
	}
	<-cc.checker.WaitReady()

	go func() {
		for i := 0; i < cc.conf.Requests; i++ {
			cc.queue <- true
		}
	}()
}

func (cc *ConcurrentChecker) doCheck() {
	err := cc.checker.CheckAddr(cc.conf.Addr, cc.conf.Timeout)
	cc.counter.Inc(CRequest)
	switch err {
	case tcpshaker.ErrTimeout:
		cc.counter.Inc(CErrTimeout)
	case nil:
		cc.counter.Inc(CSucceed)
	default:
		if cc.conf.Verbose {
			log.Println(err)
		}
		if _, ok := err.(*tcpshaker.ErrConnect); ok {
			cc.counter.Inc(CErrConnect)
		} else {
			cc.counter.Inc(CErrOther)
		}
	}
}

// Wait returns a chan which is closed when all checks are done.
func (cc *ConcurrentChecker) Wait() chan bool {
	c := make(chan bool)
	go func() {
		cc.wg.Wait()
		close(c)
	}()
	return c
}

// Stop stops the workers.
func (cc *ConcurrentChecker) Stop() {
	close(cc.closed)
}

func (cc *ConcurrentChecker) worker() {
	for {
		select {
		case <-cc.queue:
			cc.doCheck()
			cc.wg.Done()
		case <-cc.closed:
			return
		}
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	log.SetFlags(0)
	conf := parseConfig()

	if conf.Verbose {
		log.Printf(`Checking %s with the following configurations:
    Timeout: %s
   Requests: %d
Concurrency: %d

`, conf.Addr, conf.Timeout, conf.Requests, conf.Concurrency)
	}

	checker := NewConcurrentChecker(conf)
	defer checker.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	startedAt := time.Now()

	checker.Launch(ctx)
	select {
	case <-ctx.Done():
	case <-checker.Wait():
	}
	duration := time.Since(startedAt)
	stop()

	log.Printf("Finished %d/%d checks in %s\n", checker.Count(CRequest), conf.Requests, duration)
	log.Printf("  Succeed: %d\n", checker.Count(CSucceed))
	log.Printf("  Errors: connect %d, timeout %d, other %d\n", checker.Count(CErrConnect), checker.Count(CErrTimeout), checker.Count(CErrOther))
}
