package nbd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

type serviceSocket struct {
	descriptor
}

func (ss *serviceSocket) server(ctx context.Context) error {
	fmt.Println("starting server")
	f := os.NewFile(uintptr(ss.descriptor), "unix")
	mem := Memory{
		data: make([]byte, diskSize),
	}
	for {
		req := request(make([]byte, 28))
		if n, err := f.Read(req); err != nil || n != 28 {
			// if no error but not bytes, the connection was closed, exit the server
			if errors.Is(err, io.EOF) && n == 0 {
				return nil
			}
			return fmt.Errorf("local nbd server could not read request, got %d bytes: %w", n, err)
		}
		if req.magic() != nbd_REQUEST_MAGIC {
			return fmt.Errorf("Fatal error: received packet with wrong Magic number")
		}
		var rep *reply
		var replyData []byte
		switch req.command() {
		case nbd_CMD_DISC:
			mem.Disconnect()
			return nil
		case nbd_CMD_READ:
			rep = newReply(req.handle())
			replyData = make([]byte, req.len())
			if err := mem.ReadAt(replyData, req.offset()); err != nil {
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
		case nbd_CMD_TRIM:
			rep = newReply(req.handle())
			if err := mem.TrimAt(req.offset(), req.len()); err != nil {
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
		}
	}
}
