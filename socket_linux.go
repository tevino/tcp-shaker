package tcp

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/tevino/tcp-shaker/internal"
	"golang.org/x/sys/unix"
)

const maxEpollEvents = 32

// createSocketWithOptions creates a socket with specified options
func createSocketWithOptions(family int, zeroLinger bool, mark int) (fd int, err error) {
	// Create socket
	fd, err = _createNonBlockingSocket(family)
	if err != nil {
		return
	}

	if zeroLinger {
		if err = _setZeroLinger(fd); err != nil {
			_ = unix.Close(fd)
			return
		}
	}

	if mark != 0 {
		if err = _setMark(fd, mark); err != nil {
			_ = unix.Close(fd)
			return
		}
	}

	return
}

// createNonBlockingSocket creates a non-blocking socket with necessary options all set.
func _createNonBlockingSocket(family int) (int, error) {
	// Create socket
	fd, err := _createSocket(family)
	if err != nil {
		return 0, err
	}
	// Set necessary options
	err = _setSockOpts(fd)
	if err != nil {
		unix.Close(fd)
	}
	return fd, err
}

// createSocket creates a socket with CloseOnExec set
func _createSocket(family int) (int, error) {
	fd, err := unix.Socket(family, unix.SOCK_STREAM, 0)
	unix.CloseOnExec(fd)
	return fd, err
}

// setSockOpts sets SOCK_NONBLOCK and TCP_QUICKACK for given fd
func _setSockOpts(fd int) error {
	err := unix.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_QUICKACK, 0)
}

var zeroLinger = unix.Linger{Onoff: 1, Linger: 0}

// setLinger sets SO_Linger with 0 timeout to given fd
func _setZeroLinger(fd int) error {
	return unix.SetsockoptLinger(fd, unix.SOL_SOCKET, unix.SO_LINGER, &zeroLinger)
}

// setMark sets SO_MARK for given fd (Linux only)
func _setMark(fd int, mark int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_MARK, mark)
}

func createPoller() (fd int, err error) {
	fd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		err = os.NewSyscallError("epoll_create1", err)
	}
	return fd, err
}

// registerEvents registers given fd with read and write events.
func registerEvents(pollerFd int, fd int) error {
	var event unix.EpollEvent
	event.Events = unix.EPOLLOUT | unix.EPOLLIN | unix.EPOLLET
	event.Fd = int32(fd)
	if err := unix.EpollCtl(pollerFd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		return os.NewSyscallError(fmt.Sprintf("epoll_ctl(%d, ADD, %d, ...)", pollerFd, fd), err)
	}
	return nil
}

func pollEvents(pollerFd int, timeout time.Duration) ([]internal.Event, error) {
	var timeoutMS = int(timeout.Nanoseconds() / 1000000)
	var epollEvents [maxEpollEvents]unix.EpollEvent
	nEvents, err := unix.EpollWait(pollerFd, epollEvents[:], timeoutMS)
	if err != nil {
		if err == unix.EINTR {
			return nil, nil
		}
		return nil, os.NewSyscallError("epoll_wait", err)
	}

	var events = make([]internal.Event, 0, nEvents)

	for i := 0; i < nEvents; i++ {
		var fd = int(epollEvents[i].Fd)
		var evt = internal.Event{Fd: fd, Err: nil}

		errCode, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
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
		if runtime.GOOS == "solaris" { //nolint:staticcheck
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
