package tcp

import (
	"net"
	"runtime"

	"golang.org/x/sys/unix"
)

// parseSockAddr resolves given addr to unix.Sockaddr
func parseSockAddr(addr string) (sAddr unix.Sockaddr, family int, err error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	if ip := tAddr.IP.To4(); ip != nil {
		var addr4 [net.IPv4len]byte
		copy(addr4[:], ip)
		sAddr = &unix.SockaddrInet4{Port: tAddr.Port, Addr: addr4}
		family = unix.AF_INET
		return
	}

	if ip := tAddr.IP.To16(); ip != nil {
		var addr16 [net.IPv6len]byte
		copy(addr16[:], ip)
		sAddr = &unix.SockaddrInet6{Port: tAddr.Port, Addr: addr16}
		family = unix.AF_INET6
		return
	}

	err = &net.AddrError{
		Err:  "unsupported address family",
		Addr: tAddr.IP.String(),
	}
	return
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
