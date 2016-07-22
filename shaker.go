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
// Shaker's methods may be called by multiple goroutines simultaneously.
package tcp

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const maxEpollEvents = 32

// Shaker contains an epoll instance for TCP handshake checking
type Shaker struct {
	sync.RWMutex
	epollFd int
}

// InitShaker creates inner epoll instance, call this before anything else
func (s *Shaker) InitShaker() error {
	var err error
	s.Lock()
	defer s.Unlock()
	// Check if we already initialized
	if s.epollFd > 0 {
		return nil
	}
	// Create epoll instance
	s.epollFd, err = syscall.EpollCreate1(0)
	if err != nil {
		return os.NewSyscallError("epoll_create1", err)
	}
	return nil
}

// TestAddr performs a TCP check with given TCP address and timeout
// A successful check will result in nil error
// ErrTimeout is returned if timeout
// Note: timeout includes domain resolving
func (s *Shaker) TestAddr(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	// Parse address
	rAddr, err := parseSockAddr(addr)
	if err != nil {
		return err
	}
	// Create socket
	fd, err := createSocket()
	if err != nil {
		return err
	}
	defer syscall.Close(fd)
	// Set necessary options
	if err = setSockopts(fd); err != nil {
		return err
	}
	// Connect to the address
	if err = s.doConnect(fd, rAddr); err != nil {
		return err
	}
	// Check if the deadline was hit
	if reached(deadline) {
		return ErrTimeout
	}
	// Register to epoll for later error checking
	s.registerFd(fd)
	timeoutMS := int(timeout.Nanoseconds() / 1000000)
	// Check for connect error
	for {
		succeed, err := s.waitForConnected(fd, timeoutMS)
		// Check if the deadline was hit
		if reached(deadline) {
			return ErrTimeout
		}
		if err != nil {
			return err
		}
		if succeed {
			return nil
		}
	}
}

// Ready returns a bool indicates whether the Shaker is ready for use
func (s *Shaker) Ready() bool {
	s.RLock()
	defer s.RUnlock()
	return s.epollFd > 0
}

// EpollFd returns the inner fd of epoll instance
// Note: Use this only when you really know what you are doing
func (s *Shaker) EpollFd() int {
	s.RLock()
	defer s.RUnlock()
	return s.epollFd
}

// Close closes the inner epoll fd
// InitShaker needs to be called before reuse of the closed shaker
func (s *Shaker) Close() error {
	s.Lock()
	defer s.Unlock()
	if s.epollFd > 0 {
		err := syscall.Close(s.epollFd)
		s.epollFd = 0
		return err
	}
	return nil
}

// registerFd registers given fd to epoll with EPOLLOUT
func (s *Shaker) registerFd(fd int) error {
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLOUT
	event.Fd = int32(fd)
	s.RLock()
	defer s.RUnlock()
	if err := syscall.EpollCtl(s.epollFd, syscall.EPOLL_CTL_ADD, fd, &event); err != nil {
		return os.NewSyscallError("epoll_ctl", err)
	}
	return nil
}

// waitForConnected waits for epoll event of given fd with given timeout
// The boolean returned indicates whether the previous connect is successful
func (s *Shaker) waitForConnected(fd int, timeoutMS int) (bool, error) {
	var events [maxEpollEvents]syscall.EpollEvent
	s.RLock()
	epollFd := s.epollFd
	if epollFd <= 0 {
		return false, ErrNotInitialized
	}
	s.RUnlock()
	nevents, err := syscall.EpollWait(epollFd, events[:], timeoutMS)
	if err != nil {
		return false, os.NewSyscallError("epoll_wait", err)
	}

	for ev := 0; ev < nevents; ev++ {
		if int(events[ev].Fd) == fd {
			errCode, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
			if err != nil {
				return false, os.NewSyscallError("getsockopt", err)
			}
			if errCode != 0 {
				return false, newErrConnect(errCode)
			}
			return true, nil
		}
	}
	return false, nil
}

// doConnect calls the connect syscall with some error handled
func (s *Shaker) doConnect(fd int, addr syscall.Sockaddr) error {
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
