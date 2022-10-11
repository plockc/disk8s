package nbd

import (
	"fmt"
	"log"
)

type Memory struct {
	data []byte
}

func (d *Memory) ReadAt(p []byte, off uint) error {
	copy(p, d.data[off:int(off)+len(p)])
	fmt.Printf("MEMORY READ at:%d, %d bytes\n", off, len(p))
	return nil
}

func (d *Memory) WriteAt(p []byte, off uint) error {
	copy(d.data[off:], p)
	log.Printf("MEMORY WRITE at:%d, %d bytes\n", off, len(p))
	return nil
}

func (d *Memory) Disconnect() {
	log.Println("MEMORY DISCONNECT")
}
