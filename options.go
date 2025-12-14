package tcp

import (
	"time"
)

// Options contains configuration for TCP connectivity checks
type Options struct {
	// Timeout specifies the maximum duration for the check operation
	Timeout time.Duration

	// Network specifies the network type for address resolution and connection
	// Supported values:
	//   "tcp"  - Try IPv4 first, then IPv6 (default behavior)
	//   "tcp4" - IPv4 only
	//   "tcp6" - IPv6 only
	Network string

	// ZeroLinger indicates whether to set SO_LINGER with zero timeout
	// This forces the connection to be reset immediately when closed
	ZeroLinger bool

	// Mark sets the SO_MARK socket option (Linux only)
	// This is useful for traffic marking and routing policies
	// Value of 0 means no mark is set
	Mark int
}

// DefaultOptions returns Options with default values
func DefaultOptions() Options {
	return Options{
		Timeout:    time.Second * 3,
		Network:    "tcp",
		ZeroLinger: true,
		Mark:       0, // No mark by default
	}
}

// WithTimeout sets the timeout for the operation
func (o Options) WithTimeout(timeout time.Duration) Options {
	o.Timeout = timeout
	return o
}

// WithNetwork sets the network type (tcp, tcp4, tcp6)
func (o Options) WithNetwork(network string) Options {
	o.Network = network
	return o
}

// WithZeroLinger sets the zero linger option
func (o Options) WithZeroLinger(zeroLinger bool) Options {
	o.ZeroLinger = zeroLinger
	return o
}

// WithMark sets the SO_MARK socket option (Linux only)
// This is useful for traffic marking and routing policies
func (o Options) WithMark(mark int) Options {
	o.Mark = mark
	return o
}
