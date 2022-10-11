package nbd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
)

// TODO: flag for device index
type Device struct {
	deviceIndex uint8
}

const diskSize = 10 * 1024 * 1024

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

const (
	nbd_FLAG_SEND_TRIM = (1 << 5)
)

func Driver(ctx context.Context) error {
	deviceName := "/dev/nbd0"

	// the device is like /dev/nbd0 and is used by the user as a block device
	// this code will interact with it as a device with ioctl
	fmt.Println("opening " + deviceName)
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

	shutdownDevice := func() {
		// let us try to clear out the queue before exiting, eh?
		//_ = devDeviceFd.ioctl(nbd_CLEAR_QUE, 0)
		_ = devDeviceFd.ioctl(nbd_CLEAR_SOCK, 0)
		fmt.Println("Device has been disconnected")
	}
	defer shutdownDevice()

	// the socketPair is a pair of anonymous connected unix domain socket.
	// one goes to the kernel, the other this process
	fmt.Println("opening UNIX domain socket")
	socketPair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Printf("Failed to created socketpair: %v", err)
		os.Exit(1)
	}
	closeSocketPair := func() {
		syscall.Close(socketPair[0])
		syscall.Close(socketPair[1])
		fmt.Println("Socket Pair for IOCTL has been shut down")
	}
	defer closeSocketPair()
	// the first socket in the pair went to the devDevice, the second we'll use in the
	// nbd server
	ss := serviceSocket{descriptor: descriptor(socketPair[1])}

	// one option would have been to send NBD_BLOCKSIZE and NBD_SIZE_BLOCKS, but we can also
	// just do NBD_SET_SIZE with bytes
	fmt.Printf("Setting disk size to %dMB\n", diskSize/1024/1024)
	_ = devDeviceFd.ioctl(nbd_SET_SIZE, uintptr(diskSize))

	// set the request / reply socket on /dev/nbd*
	fmt.Println("Setting domain socket")
	if err := devDeviceFd.ioctl(nbd_SET_SOCK, uintptr(socketPair[0])); err != nil {
		return fmt.Errorf("failed to give the kernel its UNIX domain socket: %w", err)
	}

	// tell kernel that TRIM is supported
	fmt.Println("Send Flags, indicating TRIM is supported")
	devDeviceFd.ioctl(nbd_SET_FLAGS, nbd_FLAG_SEND_TRIM)

	fmt.Println("Starting client...")
	wg := sync.WaitGroup{}
	wg.Add(1)
	var clientErr error
	go func() {
		defer wg.Done()
		// this will block until the kernel receives close
		fmt.Println("sending DO_IT")
		clientErr = devDeviceFd.ioctl(nbd_DO_IT, 0)
		fmt.Println("DO_IT is done, shutting down, client error status:", clientErr)
		ss.shutdown()
	}()

	// this will exit after receiving a disconnect request or there was an error
	serverErr := ss.server(ctx)

	fmt.Println("server exited, waiting for /dev/nbd* client to finish, err status:", serverErr)
	// wait for routines to complete before exiting
	wg.Wait()

	if serverErr != nil {
		return serverErr
	}
	return clientErr
}
