package tcp

import "syscall"

// setSockOpts sets SOCK_NONBLOCK and TCP_QUICKACK for given fd
func setSockOpts(fd int) error {
	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_TCP, syscall.TCP_QUICKACK, 0)
}
