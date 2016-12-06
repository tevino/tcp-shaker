// +build linux

package tcp

import (
	"net"
	"syscall"
)

// parseSockAddr resolves given addr to syscall.Sockaddr
func parseSockAddr(addr string) (syscall.Sockaddr, error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	var addr4 [4]byte
	if tAddr.IP != nil {
		copy(addr4[:], tAddr.IP.To4()) // copy last 4 bytes of slice to array
	}
	return &syscall.SockaddrInet4{Port: tAddr.Port, Addr: addr4}, nil
}

// createSocket creates a socket with CloseOnExec set
func createSocket() (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	syscall.CloseOnExec(fd)
	return fd, err
}

// setSockOpts sets SOCK_NONBLOCK and TCP_QUICKACK for given fd
func setSockOpts(fd int) error {
	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_TCP, syscall.TCP_QUICKACK, 0)
}

var zeroLinger = syscall.Linger{Onoff: 1, Linger: 0}

// setLinger sets SO_Linger with 0 timeout to given fd
func setZeroLinger(fd int) error {
	return syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &zeroLinger)
}
