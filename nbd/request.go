package nbd

import "encoding/binary"

const (
	nbd_REQUEST_MAGIC = 0x25609513
)

type request []byte

func (r request) magic() uint32 {
	return binary.BigEndian.Uint32(r[0:4])
}

type command uint32

const (
	nbd_CMD_READ  command = 0
	nbd_CMD_WRITE command = 1
	nbd_CMD_DISC  command = 2
	nbd_CMD_FLUSH command = 3
	nbd_CMD_TRIM  command = 4
)

func (r request) command() command {
	return command(binary.BigEndian.Uint32(r[4:8]))
}
func (r request) handle() uint64 {
	return binary.BigEndian.Uint64(r[8:16])
}
func (r request) offset() uint {
	return uint(binary.BigEndian.Uint64(r[16:24]))
}
func (r request) len() uint32 {
	return binary.BigEndian.Uint32(r[24:28])
}
