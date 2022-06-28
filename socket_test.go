package tcp

import (
	"bytes"
	"net"
	"testing"

	"golang.org/x/sys/unix"
)

func TestParseSockAddr(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, _, err := parseSockAddr("127.0.0.1")
		assert(t, err != nil)
	})

	t.Run("ipv4", func(t *testing.T) {
		sAddr, family, err := parseSockAddr("127.0.0.1:8080")
		assert(t, err == nil)
		assert(t, unix.AF_INET == family)
		sAddr4, ok := sAddr.(*unix.SockaddrInet4)
		assert(t, ok)
		ipEqual := bytes.Equal(
			sAddr4.Addr[:],
			net.ParseIP("127.0.0.1").To4(),
		)
		assert(t, ipEqual)
		assert(t, sAddr4.Port == 8080)
	})

	t.Run("ipv6", func(t *testing.T) {
		sAddr, family, err := parseSockAddr("[fdbd:dc03:ff:1:1:25:25:225]:8080")
		assert(t, err == nil)
		assert(t, unix.AF_INET6 == family)
		sAddr6, ok := sAddr.(*unix.SockaddrInet6)
		assert(t, ok)
		ipEqual := bytes.Equal(
			sAddr6.Addr[:],
			net.ParseIP("fdbd:dc03:ff:1:1:25:25:225").To16(),
		)
		assert(t, ipEqual)
		assert(t, sAddr6.Port == 8080)
	})
}
