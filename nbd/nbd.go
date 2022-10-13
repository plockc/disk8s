package nbd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
)

// TODO: flag for device index
type Device struct {
	deviceIndex uint8
}

// operation is a ioctl operation to manage the network block device
type operation uintptr

const (
	nbd_SET_SOCK   operation = (0xab<<8 | 0)
	nbd_SET_SIZE   operation = (0xab<<8 | 2)
	nbd_DO_IT      operation = (0xab<<8 | 3)
	nbd_CLEAR_SOCK operation = (0xab<<8 | 4)
	nbd_CLEAR_QUE  operation = (0xab<<8 | 5)
	nbd_DISCONNECT operation = (0xab<<8 | 8)
	nbd_SET_FLAGS  operation = (0xab<<8 | 10)
)

func NewDomainSocketClient(ctx context.Context, deviceName string, domainSockets chan<- uintptr) error {
	// the socketPair is a pair of anonymous connected unix domain socket.
	// one goes to the kernel, the other this process
	fmt.Println("opening UNIX domain sockets for client and server")
	socketPair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Printf("Failed to create socketpair: %v", err)
		os.Exit(1)
	}
	closeSocketPair := func() {
		syscall.Close(socketPair[0])
		syscall.Close(socketPair[1])
		fmt.Println("domain sockets for communicating with kernel are closed")
	}
	defer closeSocketPair()

	// the first socket in the pair went to the devDevice, the second we'll use in the
	// nbd server
	domainSockets <- uintptr(socketPair[1])
	close(domainSockets)

	return Client(ctx, deviceName, 10*1024*1024, 0, uintptr(socketPair[0]))
}

func NewTcpClient(ctx context.Context, deviceName string, port int) error {
	err := withTcpConn(port, func(c *net.TCPConn) error {
		f, err := c.File()
		if err != nil {
			return err
		}
		diskSize, flags, err := readGreeting(f)
		if err != nil {
			return err
		}
		defer f.Close()
		return Client(ctx, deviceName, diskSize, flags, f.Fd())
	})
	return err
}

func readGreeting(in io.Reader) (diskSize uint64, flags uint32, err error) {
	greeting := make([]byte, 152)
	n, err := in.Read(greeting)
	if err != nil || n != 152 {
		return 0, 0, fmt.Errorf("Client Failed to read greeting during negotiation, read %d/152 bytes and error: %w", n, err)
	}

	if binary.BigEndian.Uint64([]byte("NBDMAGIC")) != binary.BigEndian.Uint64(greeting[0:8]) {
		return 0, 0, fmt.Errorf("missing NBDMAGIC from server greeting")
	}
	if nbd_CLISERV_MAGIC != binary.BigEndian.Uint64(greeting[8:16]) {
		return 0, 0, fmt.Errorf("missing CLISERV magic from server greeting")
	}
	diskSize = binary.BigEndian.Uint64(greeting[16:24])
	flags = binary.BigEndian.Uint32(greeting[24:48])
	return
}

func withTcpConn(port int, f func(*net.TCPConn) error) error {
	fmt.Println("opening TCP connection to server")
	conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{Port: port})
	if err != nil {
		return err
	}
	defer conn.Close()
	err = f(conn)
	return err
}

func Client(ctx context.Context, deviceName string, diskSize uint64, flags uint32, socket uintptr) error {
	// the device is like /dev/nbd0 and is used by the user as a block device
	// this code will interact with it as a device with ioctl
	fmt.Println("Starting nbd device", deviceName, "...")
	devDeviceFile, err := os.OpenFile(deviceName, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf(
			"Cannot open \"%s\". Make sure the 'nbd' kernel module is loaded: %w",
			deviceName, err,
		)
	}
	defer func() {
		if err := devDeviceFile.Close(); err != nil {
			fmt.Println("failed to properly close the device: %w", err)
		}
		fmt.Println("Device Closed, exiting")
	}()
	devDeviceFd := descriptor(devDeviceFile.Fd())

	shutdownOnce := sync.Once{}
	shutdownDevice := func() {
		shutdownOnce.Do(func() {
			// let us try to clear out the queue before exiting, eh?
			_ = devDeviceFd.ioctl(nbd_DISCONNECT, 0)
			_ = devDeviceFd.ioctl(nbd_CLEAR_QUE, 0)
			_ = devDeviceFd.ioctl(nbd_CLEAR_SOCK, 0)
			fmt.Println("Client Device has been disconnected")
		})
	}
	defer shutdownDevice()

	// set the request / reply socket on /dev/nbd*
	fmt.Println("Setting socket after clearing prior socket (in case of prior crash)")
	_ = devDeviceFd.ioctl(nbd_CLEAR_QUE, 0)
	_ = devDeviceFd.ioctl(nbd_CLEAR_SOCK, 0)

	// one option would have been to send NBD_BLOCKSIZE and NBD_SIZE_BLOCKS, but we can also
	// just do NBD_SET_SIZE with bytes
	fmt.Printf("Setting disk size to %dMB\n", diskSize/1024/1024)
	_ = devDeviceFd.ioctl(nbd_SET_SIZE, uintptr(diskSize))

	if err := devDeviceFd.ioctl(nbd_SET_SOCK, uintptr(socket)); err != nil {
		return fmt.Errorf("failed to give the kernel its UNIX domain socket: %w", err)
	}

	fmt.Println("Send Flags")
	devDeviceFd.ioctl(nbd_SET_FLAGS, uintptr(flags))

	// handle external shutdown or internal shutdown
	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		fmt.Println("begin gracefully shutting down device client...")
		shutdownDevice()
	}()

	var clientErr error
	// this will block until the kernel receives close
	fmt.Println("Client signalling kernel to start handling block device")
	clientErr = devDeviceFd.ioctl(nbd_DO_IT, 0)
	if clientErr != nil {
		if errNo, ok := errors.Unwrap(clientErr).(syscall.Errno); ok && errNo == syscall.EBUSY {
			fmt.Println("is the nbd device mounted?", clientErr)
		} else {
			fmt.Println("nbd device client is done with error", clientErr)
		}
	} else {
		fmt.Println("kernel has released the client")
	}

	return clientErr
}
