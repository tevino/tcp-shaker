package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	tcp "github.com/tevino/tcp-shaker"
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

// ConcurrentChecker wrapper of tcp.Checker with concurrent checking capacibilities.
type ConcurrentChecker struct {
	conf    *Config
	counter *Counter
	checker *tcp.Checker
	queue   chan bool
	closed  chan bool
	wg      sync.WaitGroup
}

// NewConcurrentChecker creates a checker.
func NewConcurrentChecker(conf *Config) *ConcurrentChecker {
	return &ConcurrentChecker{
		conf:    conf,
		counter: NewCounter(CRequest, CSucceed, CErrConnect, CErrTimeout, CErrOther),
		checker: tcp.NewChecker(),
		queue:   make(chan bool),
		closed:  make(chan bool),
	}

}

// Count returns the count of given ID.
func (cc *ConcurrentChecker) Count(i int) uint64 {
	return cc.counter.Count(i)
}

// Launch initialize the checker.
func (cc *ConcurrentChecker) Launch(ctx context.Context) error {
	var err error
	go func() {
		err := cc.checker.CheckingLoop(ctx)
		log.Fatal("Error during checking loop: ", err)
	}()

	for i := 0; i < cc.conf.Concurrency; i++ {
		go cc.worker()
	}
	cc.wg.Add(cc.conf.Requests)

	if cc.conf.Verbose {
		fmt.Println("Waiting for checker to be ready")
	}
	<-cc.checker.WaitReady()

	go func() {
		for i := 0; i < cc.conf.Requests; i++ {
			cc.queue <- true
		}
	}()
	return err
}

func (cc *ConcurrentChecker) doCheck() {
	err := cc.checker.CheckAddr(cc.conf.Addr, cc.conf.Timeout)
	cc.counter.Inc(CRequest)
	switch err {
	case tcp.ErrTimeout:
		cc.counter.Inc(CErrTimeout)
	case nil:
		cc.counter.Inc(CSucceed)
	default:
		if cc.conf.Verbose {
			fmt.Println(err)
		}
		if _, ok := err.(*tcp.ErrConnect); ok {
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
