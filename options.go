package tcp

import (
	"time"
)

// Options contains configuration for TCP connectivity checks
type Options struct {
	// Timeout specifies the maximum duration for the check operation
	Timeout time.Duration

	// Network specifies the network type for address resolution and connection.
	//
	// Supported values:
	//   "tcp4"   IPv4 only.
	//   "tcp6"   IPv6 only.
	//   "tcp"    Default. Accepts both IPv4 and IPv6.
	//
	//   - IP literal:      the address family matches the literal.
	//   - Hostname+tcp4/6: only the requested record type (A/AAAA)
	//                      is resolved.
	//   - Hostname+tcp:    resolved to whichever IP the system DNS
	//                      resolver returns first (as of Go 1.26).
	//
	// Only that one address is tried; there is no fallback between
	// IPv4 and IPv6.
	Network string

	// ZeroLinger indicates whether to set SO_LINGER with zero timeout
	// This forces the connection to be reset immediately when closed
	ZeroLinger bool
}

// DefaultOptions returns Options with default values
func DefaultOptions() Options {
	return Options{
		Timeout:    time.Second * 3,
		Network:    "tcp",
		ZeroLinger: true,
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
