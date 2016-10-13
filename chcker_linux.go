package tcp

import (
	"os"
	"syscall"
)

// InitChecker creates inner epoll instance, call this before anything else
func (s *Checker) InitChecker() error {
	var err error
	s.Lock()
	defer s.Unlock()
	// Check if we already initialized
	if s.poolFd > 0 {
		return nil
	}
	// Create epoll instance
	s.poolFd, err = syscall.EpollCreate1(0)
	if err != nil {
		return os.NewSyscallError("epoll_create1", err)
	}
	return nil
}

// registerFd registers given fd to epoll with EPOLLOUT
func (s *Checker) registerFd(fd int) error {
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLOUT
	event.Fd = int32(fd)
	s.RLock()
	defer s.RUnlock()
	if err := syscall.EpollCtl(s.poolFd, syscall.EPOLL_CTL_ADD, fd, &event); err != nil {
		return os.NewSyscallError("epoll_ctl", err)
	}
	return nil
}

// waitForConnected waits for epoll event of given fd with given timeout
// The boolean returned indicates whether the previous connect is successful
func (s *Checker) waitForConnected(fd int, timeoutMS int) (bool, error) {
	var events [maxEpollEvents]syscall.EpollEvent
	s.RLock()
	epollFd := s.poolFd
	if epollFd <= 0 {
		return false, ErrNotInitialized
	}
	s.RUnlock()
	nevents, err := syscall.EpollWait(epollFd, events[:], timeoutMS)
	if err != nil {
		return false, os.NewSyscallError("epoll_wait", err)
	}

	for ev := 0; ev < nevents; ev++ {
		if int(events[ev].Fd) == fd {
			errCode, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
			if err != nil {
				return false, os.NewSyscallError("getsockopt", err)
			}
			if errCode != 0 {
				return false, newErrConnect(errCode)
			}
			return true, nil
		}
	}
	return false, nil
}
