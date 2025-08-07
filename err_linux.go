package tcp

import (
	"golang.org/x/sys/unix"
)

// newErrConnect returns a ErrConnect with given error code
func newErrConnect(errCode int) *ErrConnect {
	return &ErrConnect{unix.Errno(errCode)}
}
