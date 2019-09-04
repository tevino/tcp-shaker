package tcp

import (
	"net"
	"runtime"
	"syscall"
)

// parseSockAddr resolves given addr to syscall.Sockaddr
func parseSockAddr(addr string) (syscall.Sockaddr, error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	var addr4 [net.IPv4len]byte
	if tAddr.IP == nil {
		return &syscall.SockaddrInet4{Port: tAddr.Port, Addr: addr4}, nil
	}
	if v4 := tAddr.IP.To4(); v4 != nil {
		if tAddr.IP != nil {
			copy(addr4[:], tAddr.IP.To4()) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet4{Port: tAddr.Port, Addr: addr4}, nil
	}

	var addr6 [net.IPv6len]byte
	if tAddr.IP != nil {
		copy(addr6[:], tAddr.IP.To16()) // copy last 16 bytes of slice to array
	}
	return &syscall.SockaddrInet6{Port: tAddr.Port, Addr: addr6}, nil
}

// connect calls the connect syscall with error handled.
func connect(fd int, addr syscall.Sockaddr) (success bool, err error) {
	switch serr := syscall.Connect(fd, addr); serr {
	case syscall.EALREADY, syscall.EINPROGRESS, syscall.EINTR:
		// Connection could not be made immediately but asynchronously.
		success = false
		err = nil
	case nil, syscall.EISCONN:
		// The specified socket is already connected.
		success = true
		err = nil
	case syscall.EINVAL:
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
