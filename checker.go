// +build !linux

package tcp

import (
	"net"
	"time"
)

// Checker is a fake implementation.
type Checker struct{}

// NewChecker creates a Checker, parameters are ignored.
func NewChecker(zeroLinger bool) *Checker { return &Checker{} }

// InitChecker is unnecessary on this platform.
func (s *Checker) InitChecker() error { return nil }

// CheckAddr performs a TCP check with given TCP address and timeout.
// NOTE: zeroLinger is ignored on this platform.
func (s *Checker) CheckAddr(addr string, timeout time.Duration, zeroLinger ...bool) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if conn != nil {
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
