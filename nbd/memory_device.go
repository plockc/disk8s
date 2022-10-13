package nbd

import (
	"fmt"
	"log"
)

var _ Storage = &Memory{}

type Memory struct {
	data []byte
}

func NewMemory() Storage {
	m := Memory{}
	m.data = make([]byte, m.Size())
	return &m
}

func (m *Memory) ReadAt(p []byte, off uint64) error {
	if int(off)+len(p) > len(m.data) {
		return fmt.Errorf(
			"cannot read %d bytes starting at %d with disk size %d",
			len(p), off, len(m.data),
		)
	}
	copy(p, m.data[off:int(off)+len(p)])
	log.Printf("MEMORY READ at:%d, %d bytes\n", off, len(p))
	return nil
}

func (m *Memory) WriteAt(p []byte, off uint64) error {
	if int(off)+len(p) > len(m.data) {
		return fmt.Errorf(
			"cannot write %d bytes starting at %d with disk size %d",
			len(p), off, len(m.data),
		)
	}
	copy(m.data[off:], p)
	log.Printf("MEMORY WRITE at:%d, %d bytes\n", off, len(p))
	return nil
}

func (m *Memory) TrimAt(off uint64, length uint32) error {
	log.Printf("MEMORY TRIM at:%d, %d bytes\n", off, length)
	return nil
}

func (m *Memory) Size() uint64 {
	return 100 * 1024 * 1024
}

func (m *Memory) Disconnect() {
	log.Println("MEMORY DISCONNECT")
}
