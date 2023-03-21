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
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if conn != nil {
		if zeroLinger {
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

// CheckAddrWithLatency is the supplement of CheckAddr which return the handshake time duration.
// NOTE: the returned time duration only meaningful when return err is nil.
func (c *Checker) CheckAddrWithLatency(addr string, sourceAddr string, timeout time.Duration) (time.Duration, error) {
	return c.CheckAddrZeroLingerWithLatency(addr, timeout, c.zeroLinger)
}

// CheckAddrZeroLingerWithLatency is CheckAddrWithLatency with a zeroLinger parameter.
func (c *Checker) CheckAddrZeroLingerWithLatency(addr string, sourceAddr string, timeout time.Duration, zeroLinger bool) (time.Duration, error) {
	// Connect started at
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	duration := time.Now().Sub(start)
	if conn != nil {
		if zeroLinger {
			// Simply ignore the error since this is a fake implementation.
			conn.(*net.TCPConn).SetLinger(0)
		}
		conn.Close()
	}
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Timeout() {
			return duration, ErrTimeout
		}
	}
	return duration, err
}

// IsReady is always true on this platform.
func (c *Checker) IsReady() bool { return true }

// WaitReady returns a closed chan on this platform.
func (c *Checker) WaitReady() <-chan struct{} {
	return c.isReady
}

// Close is unnecessary on this platform.
func (c *Checker) Close() error { return nil }
