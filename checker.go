// +build !linux

package tcp

import (
	"net"
	"time"
)

// Checker is a fake implementation.
type Checker struct {
	zeroLinger bool
}

// NewChecker creates a Checker with linger set to zero or not.
func NewChecker(zeroLinger bool) *Checker {
	return &Checker{zeroLinger: zeroLinger}
}

// InitChecker is unnecessary on this platform.
func (s *Checker) InitChecker() error { return nil }

// CheckAddr performs a TCP check with given TCP address and timeout.
// NOTE: zeroLinger is ignored on non-POSIX operating systems because
// net.TCPConn.SetLinger is only implemented in src/net/sockopt_posix.go.
func (s *Checker) CheckAddr(addr string, timeout time.Duration, zeroLinger ...bool) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if conn != nil {
		if (len(zeroLinger) > 0 && zeroLinger[0]) || s.zeroLinger {
			// Simply ignore the error since this is a fake implementation.
			conn.(*net.TCPConn).SetLinger(0)
		}
		conn.Close()
	}
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Timeout() {
			return ErrTimeout
		}
	}
	return err
}

// Ready is always true on this platform.
func (s *Checker) Ready() bool { return true }

// Close is unnecessary on this platform.
func (s *Checker) Close() error { return nil }
