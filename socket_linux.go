package tcp

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

const maxEpollEvents = 32

// createSocket creates a socket with necessary options set.
func createSocketZeroLinger(zeroLinger bool, ipv6 bool) (fd int, err error) {
	// Create socket
	fd, err = _createNonBlockingSocket(ipv6)
	if err == nil {
		if zeroLinger {
			err = _setZeroLinger(fd)
		}
	}
	return
}

// createNonBlockingSocket creates a non-blocking socket with necessary options all set.
func _createNonBlockingSocket(ipv6 bool) (int, error) {
	// Create socket
	fd, err := _createSocket(ipv6)
	if err != nil {
		return 0, err
	}
	// Set necessary options
	err = _setSockOpts(fd)
	if err != nil {
		syscall.Close(fd)
	}
	return fd, err
}

// createSocket creates a socket with CloseOnExec set
func _createSocket(ipv6 bool) ( int,  error) {
	domain := syscall.AF_INET
	if ipv6 {
		domain = syscall.AF_INET6
	}
	fd, err := syscall.Socket(domain, syscall.SOCK_STREAM, 0)
	syscall.CloseOnExec(fd)
	return fd, err
}

// setSockOpts sets SOCK_NONBLOCK and TCP_QUICKACK for given fd
func _setSockOpts(fd int) error {
	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_TCP, syscall.TCP_QUICKACK, 0)
}

var zeroLinger = syscall.Linger{Onoff: 1, Linger: 0}

// setLinger sets SO_Linger with 0 timeout to given fd
func _setZeroLinger(fd int) error {
	return syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &zeroLinger)
}

func createPoller() (fd int, err error) {
	fd, err = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		err = os.NewSyscallError("epoll_create1", err)
	}
	return fd, err
}

const epollET = 1 << 31

// registerEvents registers given fd with read and write events.
func registerEvents(pollerFd int, fd int) error {
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLOUT | syscall.EPOLLIN | epollET
	event.Fd = int32(fd)
	if err := syscall.EpollCtl(pollerFd, syscall.EPOLL_CTL_ADD, fd, &event); err != nil {
		return os.NewSyscallError(fmt.Sprintf("epoll_ctl(%d, ADD, %d, ...)", pollerFd, fd), err)
	}
	return nil
}

func pollEvents(pollerFd int, timeout time.Duration) ([]event, error) {
	var timeoutMS = int(timeout.Nanoseconds() / 1000000)
	var epollEvents [maxEpollEvents]syscall.EpollEvent
	nEvents, err := syscall.EpollWait(pollerFd, epollEvents[:], timeoutMS)
	if err != nil {
		if err == syscall.EINTR {
			return nil, nil
		}
		return nil, os.NewSyscallError("epoll_wait", err)
	}

	var events = make([]event, 0, nEvents)

	for i := 0; i < nEvents; i++ {
		var fd = int(epollEvents[i].Fd)
		var evt = event{Fd: fd, Err: nil}

		errCode, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
		if err != nil {
			evt.Err = os.NewSyscallError("getsockopt", err)
		}
		if errCode != 0 {
			evt.Err = newErrConnect(errCode)
		}
		events = append(events, evt)
	}
	return events, nil
}
