package tcp

import (
	"net"
	"syscall"
)

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

func createSocket() (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	syscall.CloseOnExec(fd)
	return fd, err
}

func setSockopts(fd int) error {
	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}

	linger := syscall.Linger{Onoff: 1, Linger: 0}
	syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
	return syscall.SetsockoptInt(fd, syscall.SOL_TCP, syscall.TCP_QUICKACK, 0)
}
