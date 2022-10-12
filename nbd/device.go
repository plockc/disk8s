package nbd

type Storage interface {
	Size() uint64
	ReadAt(p []byte, off uint64) error
	WriteAt(p []byte, off uint64) error
	TrimAt(off uint64, length uint32) error
	Disconnect()
}
