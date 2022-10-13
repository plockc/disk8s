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

func greeting(diskSize uint64) []byte {
	greeting := make([]byte, 152)
	copy(greeting[0:8], []byte("NBDMAGIC"))
	binary.BigEndian.PutUint64(greeting[8:16], nbd_CLISERV_MAGIC)
	binary.BigEndian.PutUint64(greeting[16:24], diskSize)
	binary.BigEndian.PutUint32(greeting[24:28], 0)
	return greeting
}

type serviceSocket struct {
	io.ReadWriter
	Storage
}

func NewDomainSocketServer(store Storage, domainSockets <-chan uintptr) error {
	fmt.Println("server has been provided a domain socket")
	var lastError error
	for domainSocketDescriptor := range domainSockets {
		service := serviceSocket{
			ReadWriter: os.NewFile(domainSocketDescriptor, "unix"),
			Storage:    store,
		}
		lastError = service.server()
	}
	return lastError
}

func NewTCPSocketServer(ctx context.Context, store Storage, port int) error {
	// listen for connections
	server, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	fmt.Println("Listening port", port)
	defer server.Close()

	// handle external shutdown or internal shutdown
	listenCtx, listenCancel := context.WithCancel(ctx)
	defer listenCancel()
	go func() {
		<-listenCtx.Done()
		fmt.Println("begin gracefully shutting down server...")
		server.Close()
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			// TODO: fix
			if listenCtx.Err() != nil && !errors.Is(listenCtx.Err(), context.Canceled) {
				return nil
			}
			return err
		}

		connCtx, connCancel := context.WithCancel(listenCtx)
		// pass in the connCtx and connCancel to avoid race with next loop iter
		go func(c net.Conn, cancel func()) {
			select {
			case <-connCtx.Done():
				fmt.Println("closing server connection")
			case <-listenCtx.Done():
				fmt.Println("closing server connection because listener closed")
				cancel()
			}
			conn.Close()
		}(conn, connCancel)

		fmt.Println("connection accepted, sending greeting")
		greet := greeting(diskSize)
		if n, err := conn.Write(greet); err != nil || n != 152 {
			fmt.Println("Failed to write greeting to client during negotiation, wrote", n, "of", len(greet), "bytes and error:", err)
		} else {
			if err := (serviceSocket{conn, store}).server(); err != nil {
				fmt.Println("Server connection exited with ERROR:", err)
			} else {
				fmt.Println("Server handler exited with no error")
			}
		}
		connCancel()
	}
}

func (ss serviceSocket) server() error {
	fmt.Println("starting server")
	store, err := NewFile()
	if err != nil {
		return err
	}
	//mem := Memory{
	//	data: make([]byte, diskSize),
	//}
	for {
		req := request(make([]byte, 28))
		if n, err := io.ReadFull(ss, req); err != nil || n != 28 {
			// if no error but not bytes, the connection was closed, exit the server
			if errors.Is(err, io.EOF) && n == 0 {
				return nil
			}
			return fmt.Errorf("nbd server could not read request, got %d bytes: %w", n, err)
		}
		if req.magic() != nbd_REQUEST_MAGIC {
			return fmt.Errorf("Fatal error: received packet with wrong Magic number")
		}
		var rep *reply
		var replyData []byte
		switch req.command() {
		case nbd_CMD_DISC:
			fmt.Println("Server is disconnecting by request of remote kernel")
			store.Release()
			return nil
		case nbd_CMD_READ:
			rep = newReply(req.handle())
			replyData = make([]byte, req.len())
			if err := store.ReadAt(replyData, req.offset()); err != nil {
				log.Println("Error:", err)
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
			if err := store.WriteAt(respData, req.offset()); err != nil {
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
