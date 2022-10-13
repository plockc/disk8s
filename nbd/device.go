package nbd

type Storage interface {
	ReadAt(p []byte, off uint64) error
	WriteAt(p []byte, off uint64) error
	Release()
}
