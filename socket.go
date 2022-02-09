package tcp

import (
	"errors"
	"net"
	"runtime"

	"golang.org/x/sys/unix"
)

// parseSockAddr resolves given addr to unix.Sockaddr
func parseSockAddr(addr string) (unix.Sockaddr, int, error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, 0, err
	}

	switch len(tAddr.IP) {
	case net.IPv4len:
		var addr4 [net.IPv4len]byte
		copy(addr4[:], tAddr.IP.To4())
		return &unix.SockaddrInet4{Port: tAddr.Port, Addr: addr4}, unix.AF_INET, nil
	case net.IPv6len:
		var addr16 [net.IPv6len]byte
		copy(addr16[:], tAddr.IP.To16())
		return &unix.SockaddrInet6{Port: tAddr.Port, Addr: addr16}, unix.AF_INET6, nil
	default:
		return nil, 0, errors.New("invalid addr")
	}
}

// connect calls the connect syscall with error handled.
func connect(fd int, addr unix.Sockaddr) (success bool, err error) {
	switch serr := unix.Connect(fd, addr); serr {
	case unix.EALREADY, unix.EINPROGRESS, unix.EINTR:
		// Connection could not be made immediately but asynchronously.
		success = false
		err = nil
	case nil, unix.EISCONN:
		// The specified socket is already connected.
		success = true
		err = nil
	case unix.EINVAL:
		// On Solaris we can see EINVAL if the socket has
		// already been accepted and closed by the server.
		// Treat this as a successful connection--writes to
		// the socket will see EOF.  For details and a test
		// case in C see https://golang.org/issue/6828.
		if runtime.GOOS == "solaris" {
			success = true
			err = nil
		} else {
			// error must be reported
			success = false
			err = serr
		}
	default:
		// Connect error.
		success = false
		err = serr
	}
	return success, err
}
