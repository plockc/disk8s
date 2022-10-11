package nbd

import (
	"fmt"
	"syscall"
)

type descriptor uintptr

func (fd descriptor) ioctl(op operation, arg uintptr) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(fd), uintptr(op), arg,
	)
	if errno == 0 {
		return nil
	}
	return fmt.Errorf("Failed operation %v: %w", op, errno)
}
