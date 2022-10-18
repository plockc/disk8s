package store

import (
	"context"
	"fmt"
	"log"
)

var _ Storage = &Memory{}

type Memory struct {
	data []byte
}

func NewMemory() Storage {
	m := Memory{}
	m.data = make([]byte, diskSize)
	return &m
}

func (m *Memory) ReadAt(_ context.Context, p []byte, off uint64) error {
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

func (m *Memory) WriteAt(_ context.Context, p []byte, off uint64) error {
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

func (m *Memory) Release() {
	log.Println("MEMORY DISCONNECT")
}

func (m *Memory) Size(_ context.Context) (uint64, error) {
	return uint64(len(m.data)), nil
}
