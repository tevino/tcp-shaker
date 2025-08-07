package tcp

import "sync"

type resultPipesMU struct {
	l             sync.Mutex
	fdResultPipes map[int]chan error
}

func newResultPipesMU() *resultPipesMU {
	return &resultPipesMU{fdResultPipes: make(map[int]chan error)}
}

func (r *resultPipesMU) PopResultPipe(fd int) (chan error, bool) {
	r.l.Lock()
	p, exists := r.fdResultPipes[fd]
	if exists {
		delete(r.fdResultPipes, fd)
	}
	r.l.Unlock()
	return p, exists
}

func (r *resultPipesMU) DeRegisterResultPipe(fd int) {
	r.l.Lock()
	delete(r.fdResultPipes, fd)
	r.l.Unlock()
}

func (r *resultPipesMU) RegisterResultPipe(fd int, pipe chan error) {
	// NOTE: the pipe should have been put back if c.fdResultPipes[fd] exists.
	r.l.Lock()
	r.fdResultPipes[fd] = pipe
	r.l.Unlock()
}
