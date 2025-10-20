//go:build unix

package tcp

import (
	"net"

	"golang.org/x/sys/unix"
)

// parseSockAddr resolves given addr to unix.Sockaddr
func parseSockAddr(addr string) (sAddr unix.Sockaddr, family int, err error) {
	return parseSockAddrWithNetwork(addr, "tcp")
}

// parseSockAddrWithNetwork resolves given addr to unix.Sockaddr with specified network
func parseSockAddrWithNetwork(addr, network string) (sAddr unix.Sockaddr, family int, err error) {
	tAddr, err := net.ResolveTCPAddr(network, addr)
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
