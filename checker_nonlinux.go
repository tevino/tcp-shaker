//go:build !linux
// +build !linux

package tcp

import (
	"context"
	"net"
	"time"
)

// Checker is a fake implementation.
type Checker struct {
	zeroLinger bool
	isReady    chan struct{}
}

// NewChecker creates a Checker with linger set to zero.
func NewChecker() *Checker {
	return NewCheckerZeroLinger(true)
}

// NewCheckerZeroLinger creates a Checker with zeroLinger set to given value.
func NewCheckerZeroLinger(zeroLinger bool) *Checker {
	isReady := make(chan struct{})
	close(isReady)
	return &Checker{zeroLinger: zeroLinger, isReady: isReady}
}

// CheckingLoop is unnecessary on this platform.
func (c *Checker) CheckingLoop(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// CheckAddr performs a TCP check with given TCP address and timeout.
// NOTE: zeroLinger is ignored on non-POSIX operating systems because
// net.TCPConn.SetLinger is only implemented in src/net/sockopt_posix.go.
func (c *Checker) CheckAddr(addr string, timeout time.Duration) error {
	return c.CheckAddrZeroLinger(addr, timeout, c.zeroLinger)
}

// CheckAddrZeroLinger is CheckerAddr with a zeroLinger parameter.
func (c *Checker) CheckAddrZeroLinger(addr string, timeout time.Duration, zeroLinger bool) error {
	opts := DefaultOptions().WithTimeout(timeout).WithZeroLinger(zeroLinger)
	return c.CheckAddrWithOptions(addr, opts)
}

// CheckAddrWithOptions performs a TCP check with given address and options.
// NOTE: zeroLinger is ignored on non-POSIX operating systems because
// net.TCPConn.SetLinger is only implemented in src/net/sockopt_posix.go.
func (c *Checker) CheckAddrWithOptions(addr string, opts Options) error {
	conn, err := net.DialTimeout(opts.Network, addr, opts.Timeout)
	if conn != nil {
		if opts.ZeroLinger {
			// Simply ignore the error since this is a fake implementation.
			_ = conn.(*net.TCPConn).SetLinger(0)
		}
		_ = conn.Close()
	}
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Timeout() {
			return ErrTimeout
		}
	}
	return err
}

// IsReady is always true on this platform.
func (c *Checker) IsReady() bool { return true }

// WaitReady returns a closed chan on this platform.
func (c *Checker) WaitReady() <-chan struct{} {
	return c.isReady
}

// Close is unnecessary on this platform.
func (c *Checker) Close() error { return nil }
