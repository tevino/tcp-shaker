// +build darwin dragonfly freebsd netbsd openbsd

package tcp

import "syscall"

// setSockOpts sets SOCK_NONBLOCK for given fd
func setSockOpts(fd int) error {
	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	return nil
}
