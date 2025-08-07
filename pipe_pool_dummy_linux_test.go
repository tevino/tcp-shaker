package tcp

type pipePoolDummy struct{}

func newPipePoolDummy() *pipePoolDummy {
	return &pipePoolDummy{}
}

func (*pipePoolDummy) GetPipe() chan error {
	return make(chan error, 1)
}

func (*pipePoolDummy) PutBackPipe(pipe chan error) {}
