package tcp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tevino/tcp-shaker/internal"
	"golang.org/x/sys/unix"
)

// Checker contains an epoll instance for TCP handshake checking.
// NOTE: Ideally only one instance of Checker should be created within a process.
type Checker struct {
	pipePool    internal.PipePool
	resultPipes internal.ResultPipes
	pollerLock  sync.Mutex
	_pollerFd   int32
	zeroLinger  bool
	isReady     chan struct{}
}

// NewChecker creates a Checker with linger set to zero.
func NewChecker() *Checker {
	return NewCheckerZeroLinger(true)
}

// NewCheckerZeroLinger creates a Checker with zeroLinger set to given value.
func NewCheckerZeroLinger(zeroLinger bool) *Checker {
	return &Checker{
		pipePool:    internal.NewPipePoolSyncPool(),
		resultPipes: internal.NewResultPipesSyncMap(),
		_pollerFd:   -1,
		zeroLinger:  zeroLinger,
		isReady:     make(chan struct{}),
	}
}

// CheckingLoop must be called before anything else.
// NOTE: this function blocks until ctx got canceled.
func (c *Checker) CheckingLoop(ctx context.Context) error {
	pollerFd, err := c.createPoller()
	if err != nil {
		return fmt.Errorf("error creating poller: %w", err)
	}
	defer func() {
		_ = c.closePoller()
	}()

	c.setReady()
	defer c.resetReady()

	return c.pollingLoop(ctx, pollerFd)
}

func (c *Checker) createPoller() (int, error) {
	c.pollerLock.Lock()
	defer c.pollerLock.Unlock()

	if c.pollerFD() > 0 {
		// return if already initialized
		return -1, ErrCheckerAlreadyStarted
	}

	pollerFd, err := createPoller()
	if err != nil {
		return -1, err
	}
	c.setPollerFD(pollerFd)

	return pollerFd, nil
}

func (c *Checker) closePoller() error {
	c.pollerLock.Lock()
	defer c.pollerLock.Unlock()
	var err error
	if c.pollerFD() > 0 {
		err = unix.Close(c.pollerFD())
	}
	c.setPollerFD(-1)
	return err
}

func (c *Checker) setReady() {
	close(c.isReady)
}

func (c *Checker) resetReady() {
	c.isReady = make(chan struct{})
}

const pollerTimeout = time.Second

func (c *Checker) pollingLoop(ctx context.Context, pollerFd int) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			evts, err := pollEvents(pollerFd, pollerTimeout)
			if err != nil {
				// fatal error
				return fmt.Errorf("error during polling loop: %w", err)
			}

			c.handlePollerEvents(evts)
		}
	}
}

func (c *Checker) handlePollerEvents(evts []internal.Event) {
	for _, e := range evts {
		if pipe, exists := c.resultPipes.PopResultPipe(e.Fd); exists {
			pipe <- e.Err
		}
		// error pipe not found
		// in this case, e.Fd should have been handled in the previous event.
	}
}

func (c *Checker) pollerFD() int {
	return int(atomic.LoadInt32(&c._pollerFd))
}

func (c *Checker) setPollerFD(fd int) {
	atomic.StoreInt32(&c._pollerFd, int32(fd))
}

// CheckAddr performs a TCP check with given TCP address and timeout
// A successful check will result in nil error
// ErrTimeout is returned if timeout
// zeroLinger is an optional parameter indicating if linger should be set to zero
// for this particular connection
// Note: timeout includes domain resolving
func (c *Checker) CheckAddr(addr string, timeout time.Duration) (err error) {
	return c.CheckAddrZeroLinger(addr, timeout, c.zeroLinger)
}

// CheckAddrZeroLinger is like CheckAddr with an extra parameter indicating whether to enable zero linger.
func (c *Checker) CheckAddrZeroLinger(addr string, timeout time.Duration, zeroLinger bool) error {
	opts := DefaultOptions().WithTimeout(timeout).WithZeroLinger(zeroLinger)
	return c.CheckAddrWithOptions(addr, opts)
}

// CheckAddrWithOptions performs a TCP check with given address and options.
// A successful check will result in nil error.
// ErrTimeout is returned if timeout.
// Note: timeout includes domain resolving.
func (c *Checker) CheckAddrWithOptions(addr string, opts Options) error {
	// Set deadline
	deadline := time.Now().Add(opts.Timeout)

	// Parse address with specified network
	rAddr, family, err := parseSockAddrWithNetwork(addr, opts.Network)
	if err != nil {
		return err
	}
	// Create socket with options set
	fd, err := createSocketWithOptions(family, opts.ZeroLinger, opts.Mark)
	if err != nil {
		return err
	}
	// Socket should be closed anyway
	defer unix.Close(fd)

	// Connect to the address
	if success, cErr := connect(fd, rAddr); cErr != nil {
		// If there was an error, return it.
		return &ErrConnect{cErr}
	} else if success {
		// If the connect was successful, we are done.
		return nil
	}
	// Otherwise wait for the result of connect.

	return c.waitConnectResult(fd, time.Until((deadline)))
}

func (c *Checker) waitConnectResult(fd int, timeout time.Duration) error {
	// get a pipe of connect result
	resultPipe := c.pipePool.GetPipe()
	defer func() {
		c.resultPipes.DeRegisterResultPipe(fd)
		c.pipePool.PutBackPipe(resultPipe)
	}()

	// this must be done before registerEvents
	c.resultPipes.RegisterResultPipe(fd, resultPipe)
	// Register to epoll for later error checking
	if err := registerEvents(c.pollerFD(), fd); err != nil {
		return err
	}

	// Wait for connect result
	return c.waitPipeTimeout(resultPipe, timeout)
}

func (c *Checker) waitPipeTimeout(pipe chan error, timeout time.Duration) error {
	select {
	case ret := <-pipe:
		return ret
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// WaitReady returns a chan which is closed when the Checker is ready for use.
func (c *Checker) WaitReady() <-chan struct{} {
	return c.isReady
}

// IsReady returns a bool indicates whether the Checker is ready for use
func (c *Checker) IsReady() bool {
	return c.pollerFD() > 0
}

// PollerFd returns the inner fd of poller instance.
// NOTE: Use this only when you really know what you are doing.
func (c *Checker) PollerFd() int {
	return c.pollerFD()
}
