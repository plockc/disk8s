package nbd

import (
	"log"
)

type Memory struct {
	data []byte
}

func (d *Memory) ReadAt(p []byte, off uint64) error {
	copy(p, d.data[off:int(off)+len(p)])
	log.Printf("MEMORY READ at:%d, %d bytes\n", off, len(p))
	return nil
}

func (d *Memory) WriteAt(p []byte, off uint64) error {
	copy(d.data[off:], p)
	log.Printf("MEMORY WRITE at:%d, %d bytes\n", off, len(p))
	return nil
}

func (d *Memory) TrimAt(off uint64, length uint32) error {
	log.Printf("MEMORY TRIM at:%d, %d bytes\n", off, length)
	return nil
}

func (d *Memory) Disconnect() {
	log.Println("MEMORY DISCONNECT")
}
