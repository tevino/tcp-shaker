package tcp

import (
	"errors"
	"syscall"
)

// ErrTimeout indicates I/O timeout
var ErrTimeout = &timeoutError{}

// ErrNotInitialized occurs while the Checker is not initialized
var ErrNotInitialized = errors.New("not initialized")

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "I/O timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// ErrConnect is an error occurs while connecting to the host
// To get the detail of underlying error, lookup ErrorCode() in 'man 2 connect'
type ErrConnect struct {
	error
}

// newErrConnect returns a ErrConnect with given error code
func newErrConnect(errCode int) *ErrConnect {
	return &ErrConnect{syscall.Errno(errCode)}
}
