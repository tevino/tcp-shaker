package internal

type PipePool interface {
	GetPipe() chan error
	PutBackPipe(chan error)
}
