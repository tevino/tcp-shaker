package internal

import "sync"

type resultPipesSyncMap struct {
	sync.Map
}

func NewResultPipesSyncMap() *resultPipesSyncMap {
	return &resultPipesSyncMap{}
}

func (r *resultPipesSyncMap) PopResultPipe(fd int) (chan error, bool) {
	p, exist := r.Load(fd)
	if exist {
		r.Delete(fd)
	}
	if p != nil {
		return p.(chan error), exist
	}
	return nil, exist
}

func (r *resultPipesSyncMap) DeRegisterResultPipe(fd int) {
	r.Delete(fd)
}

func (r *resultPipesSyncMap) RegisterResultPipe(fd int, pipe chan error) {
	// NOTE: the pipe should have been put back if c.fdResultPipes[fd] exists.
	r.Store(fd, pipe)
}
