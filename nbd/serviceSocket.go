package nbd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

var greeting []byte

func init() {
	greeting := make([]byte, 152)
	copy(greeting[0:8], []byte("NBDMAGIC"))
	binary.BigEndian.PutUint64(greeting[8:16], nbd_CLISERV_MAGIC)
	binary.BigEndian.PutUint64(greeting[16:24], diskSize)
	binary.BigEndian.PutUint32(greeting[24:28], nbd_FLAG_SEND_TRIM)
}

type serviceSocket struct {
	io.ReadWriter
}

func NewDomainSocketServer(domainSockets <-chan uintptr) error {
	var lastError error
	for domainSocketDescriptor := range domainSockets {
		lastError = serviceSocket{os.NewFile(domainSocketDescriptor, "unix")}.server()
	}
	return lastError
}

func NewTCPSocketServer(ctx context.Context, port int) error {
	// listen for connections
	server, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	fmt.Println("Listening port", port)
	defer server.Close()

	// handle external shutdown or internal shutdown
	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		fmt.Println("begin gracefully shutting down server...")
		server.Close()
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			if ctx.Err() != nil && !errors.Is(ctx.Err(), context.Canceled) {
				return err
			}
			return nil
		}

		if n, err := conn.Write(greeting); err != nil || n != 152 {
			fmt.Println("Failed to write greeting to client during negotiation, wrote", n, "bytes and error:", err)
		}
		if err := (serviceSocket{conn}).server(); err != nil {
			fmt.Println("Server connection exited with ERROR:", err)
		} else {
			fmt.Println("Server connection closed")
		}
		conn.Close()
	}
}

func (ss serviceSocket) server() error {
	fmt.Println("starting server")
	mem := Memory{
		data: make([]byte, diskSize),
	}
	for {
		req := request(make([]byte, 28))
		if n, err := ss.Read(req); err != nil || n != 28 {
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
			fmt.Println("Server is disconnecting")
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
			if _, err := io.ReadFull(ss, respData); err != nil {
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
		if rep != nil {
			if n, err := ss.Write(*rep); err != nil || n != len(*rep) {
				return fmt.Errorf("failed to send reply to /dev/nbd*: %w", err)
			}
			if n, err := ss.Write(replyData); err != nil || n != len(replyData) {
				return fmt.Errorf("failed to write back data payload to /dev/nbd*: %w", err)
			}
		}
	}
}
