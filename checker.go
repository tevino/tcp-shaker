// Package tcp is used to perform TCP handshake without ACK,
// useful for health checking, HAProxy does this exactly the same.
// Which is SYN, SYN-ACK, RST.
//
// Why do I have to do this?
// Usually when you establish a TCP connection(e.g. net.Dial), these
// are the first three packets (TCP three-way handshake):
//
//	SYN:     Client -> Server
//	SYN-ACK: Server -> Client
//	ACK:     Client -> Server
//
// This package tries to avoid the last ACK when doing handshakes.
//
// By sending the last ACK, the connection is considered established.
// However as for TCP health checking the last ACK may not necessary.
// The Server could be considered alive after it sends back SYN-ACK.
//
// Benefits of avoiding the last ACK:
//
// 1. Less packets better efficiency
//
// 2. The health checking is less obvious
//
// The second one is essential, because it bothers server less.
// Usually this means the server will not notice the health checking
// traffic at all, thus the act of health checking will not be
// considered as some misbehaviour of client.
//
// Checker's methods may be called by multiple goroutines simultaneously.
package tcp

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const maxPoolEvents = 32

// Checker contains an epoll or kqueue instance for TCP handshake checking
type Checker struct {
	sync.RWMutex
	poolFd     int // epoll or kqueue instance
	zeroLinger bool
}

// NewChecker creates a Checker with linger set to zero or not
func NewChecker(zeroLinger bool) *Checker {
	return &Checker{zeroLinger: zeroLinger}
}

// createSocket creats a socket with necessary options set
func (s *Checker) createSocket(zeroLinger ...bool) (fd int, err error) {
	// Create socket
	fd, err = createSocket()
	// Set necessary options
	if err == nil {
		err = setSockOpts(fd)
	}
	// Set linger if zeroLinger or s.zeroLinger is on
	if err == nil {
		if (len(zeroLinger) > 0 && zeroLinger[0]) || s.zeroLinger {
			err = setZeroLinger(fd)
		}
	}
	return
}

// CheckAddr performs a TCP check with given TCP address and timeout
// A successful check will result in nil error
// ErrTimeout is returned if timeout
// zeroLinger is an optional parameter indicating if linger should be set to zero
// for this particular connection
// Note: timeout includes domain resolving
func (s *Checker) CheckAddr(addr string, timeout time.Duration, zeroLinger ...bool) (err error) {
	// Set deadline
	deadline := time.Now().Add(timeout)
	// Parse address
	var rAddr syscall.Sockaddr
	if rAddr, err = parseSockAddr(addr); err != nil {
		return err
	}
	// Create socket with options set
	var fd int
	if fd, err = s.createSocket(zeroLinger...); err != nil {
		return
	}
	defer func() {
		// Socket should be closed anyway
		cErr := syscall.Close(fd)
		// Error from close should be returned if no other error happened
		if err == nil {
			err = cErr
		}
	}()
	// Connect to the address
	if err = s.doConnect(fd, rAddr); err != nil {
		return
	}
	// Check if the deadline was hit
	if reached(deadline) {
		err = ErrTimeout
		return
	}
	// Register to epoll or kqueue for later error checking
	if err = s.registerFd(fd); err != nil {
		return
	}
	// Check for connect error
	var succeed bool
	var timeoutMS = int(timeout.Nanoseconds() / 1000000)
	for {
		succeed, err = s.waitForConnected(fd, timeoutMS)
		// Check if the deadline was hit
		if reached(deadline) {
			return ErrTimeout
		}
		if succeed || err != nil {
			break
		}
	}
	return
}

// Ready returns a bool indicates whether the Checker is ready for use
func (s *Checker) Ready() bool {
	s.RLock()
	defer s.RUnlock()
	return s.poolFd > 0
}

// PoolFd returns the inner fd of epoll or kqueue instance
// Note: Use this only when you really know what you are doing
func (s *Checker) PoolFd() int {
	s.RLock()
	defer s.RUnlock()
	return s.poolFd
}

// Close closes the inner epoll or kqueue fd
// InitChecker needs to be called before reuse of the closed Checker
func (s *Checker) Close() error {
	s.Lock()
	defer s.Unlock()
	if s.poolFd > 0 {
		err := syscall.Close(s.poolFd)
		s.poolFd = 0
		return err
	}
	return nil
}

// doConnect calls the connect syscall with some error handled
func (s *Checker) doConnect(fd int, addr syscall.Sockaddr) error {
	switch err := syscall.Connect(fd, addr); err {
	case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
	case nil, syscall.EISCONN:
		// already connected
	case syscall.EINVAL:
		// On Solaris we can see EINVAL if the socket has
		// already been accepted and closed by the server.
		// Treat this as a successful connection--writes to
		// the socket will see EOF.  For details and a test
		// case in C see https://golang.org/issue/6828.
		if runtime.GOOS == "solaris" {
			return nil
		}
		fallthrough
	default:
		return os.NewSyscallError("connect", err)
	}
	return nil
}

// reached tests if the given deadline was hit
func reached(deadline time.Time) bool {
	return !deadline.IsZero() && deadline.Before(time.Now())
}
