package tcp

import (
	"errors"
)

// ErrTimeout indicates I/O timeout
var ErrTimeout = &timeoutError{}

// ErrNotInitialized occurs while the Shaker is not initialized
var ErrNotInitialized = errors.New("not initialized")

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "I/O timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
