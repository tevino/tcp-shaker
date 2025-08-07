package internal

type ResultPipes interface {
	PopResultPipe(int) (chan error, bool)
	DeRegisterResultPipe(int)
	RegisterResultPipe(int, chan error)
}
