package nbd

import (
	"encoding/binary"
)

const (
	nbd_REPLY_MAGIC          = 0x67446698
	nbd_CLISERV_MAGIC uint64 = 0x00420281861253
)

type reply []byte

func newReply(handle uint64) *reply {
	r := reply(make([]byte, 16))
	binary.BigEndian.PutUint32(r[0:4], nbd_REPLY_MAGIC)
	binary.BigEndian.PutUint64(r[8:16], handle)
	return &r
}

func (r *reply) err(err uint32) {
	binary.BigEndian.PutUint32((*r)[4:8], err)
}
