package nbd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type serviceSocket struct {
	descriptor
	sync.Once
}

func (ss *serviceSocket) server(ctx context.Context) error {
	fmt.Println("starting server")
	f := os.NewFile(uintptr(ss.descriptor), "unix")
	mem := Memory{
		data: make([]byte, diskSize),
	}
	for {
		req := request(make([]byte, 28))
		fmt.Println("waiting for request")
		if n, err := f.Read(req); err != nil || n != 28 {
			return fmt.Errorf("local nbd server could not read request, got %d bytes: %w", n, err)
		}
		if req.magic() != nbd_REQUEST_MAGIC {
			return fmt.Errorf("Fatal error: received packet with wrong Magic number")
		}
		fmt.Println("received request command is", req.command())
		var rep *reply
		var replyData []byte
		switch req.command() {
		case nbd_CMD_DISC:
			mem.Disconnect()
			ss.shutdown()
			return nil
		case nbd_CMD_READ:
			rep = newReply(req.handle())
			replyData = make([]byte, req.len())
			if err := mem.ReadAt(replyData, uint(req.offset())); err != nil {
				log.Println(err)
				// Reply with an EPERM
				rep.err(1)
				replyData = nil
				break
			}
		case nbd_CMD_WRITE:
			rep = newReply(req.handle())
			respData := make([]byte, req.len())
			if _, err := io.ReadFull(f, respData); err != nil {
				return fmt.Errorf("could not read request data for a remote device write: %w", err)
			}
			if err := mem.WriteAt(respData, req.offset()); err != nil {
				log.Println("error for data written to device when writing to remote device:", err)
				rep.err(1)
			}
		default:
			fmt.Println("UNKNOWN COMMAND", req.command())
			continue
		}
		if rep == nil {
			fmt.Println("no reply")
		} else {
			if n, err := f.Write(*rep); err != nil || n != len(*rep) {
				return fmt.Errorf("failed to send reply to /dev/nbd*: %w", err)
			}
			if n, err := f.Write(replyData); err != nil || n != len(replyData) {
				return fmt.Errorf("failed to write back data payload to /dev/nbd*: %w", err)
			}
			fmt.Println("wrote reply for", req.handle(), "and data size was", len(replyData))
		}
	}
}

func (ss *serviceSocket) shutdown() {
	ss.Once.Do(func() {
		fmt.Println("Disconnecting")
		_ = ss.ioctl(nbd_DISCONNECT, 0)
	})
}
