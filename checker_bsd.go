// +build darwin dragonfly freebsd netbsd openbsd

package tcp

import (
	"errors"
	"os"
	"syscall"
)

// InitChecker creates inner kqueue instance, call this before anything else
func (s *Checker) InitChecker() error {
	var err error
	s.Lock()
	defer s.Unlock()
	// Check if we already initialized
	if s.poolFd > 0 {
		return nil
	}
	// Create kqueue instance
	s.poolFd, err = syscall.Kqueue()
	if err != nil {
		return os.NewSyscallError("kqueue", err)
	}
	return nil
}

// registerFd registers give fd to kqueue with EVFILT_READ
func (s *Checker) registerFd(fd int) error {
	event := syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_ADD,
	}
	changes := []syscall.Kevent_t{event}
	s.RLock()
	defer s.RUnlock()
	if _, err := syscall.Kevent(s.poolFd, changes, nil, nil); err != nil {
		return os.NewSyscallError("kevent", err)
	}
	return nil
}

// waitForConnected waits for epoll event of given fd with given timeout
// The boolean returned indicates whether the previous connect is successful
func (s *Checker) waitForConnected(fd int, timeoutMS int) (bool, error) {
	// events := make([]syscall.Kevent_t, 0, maxPoolEvents)
	events := make([]syscall.Kevent_t, maxPoolEvents)
	s.RLock()
	if s.poolFd <= 0 {
		return false, ErrNotInitialized
	}
	s.RUnlock()

	timeoutNS := int64(timeoutMS * 1e6)
	timeout := &syscall.Timespec{
		Sec:  timeoutNS / 1e9,
		Nsec: timeoutNS % 1e9,
	}
	n, err := syscall.Kevent(s.poolFd, nil, events, timeout)
	if err != nil {
		return false, os.NewSyscallError("kevent", err)
	}
	if n == 0 {
		return false, os.NewSyscallError("kevent", errors.New("timeout"))
	}

	for i := 0; i < n; i++ {
		if int(events[i].Ident) == fd {
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
