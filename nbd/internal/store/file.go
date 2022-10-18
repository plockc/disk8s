package store

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
)

var diskSize uint64
var diskPath string

func init() {
	sizeStr := os.Getenv("DISK_SIZE")
	if sizeStr == "" {
		diskSize = 100 * 1024 * 1024
	} else {
		var err error
		diskSize, err = strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			panic(err.Error())
		}
	}
	diskPath = os.Getenv("DISK_PATH")
	if diskPath == "" {
		diskPath = "disk8s.data"
	}
}

var _ Storage = &File{}

type File struct {
	*os.File
}

func NewFile() (Storage, error) {
	log.Printf("FILE OPEN SIZE " + strconv.Itoa(int(diskSize/1024/1024)) + "MiB")
	file, err := os.OpenFile(diskPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if uint64(info.Size()) < diskSize {
		err = file.Truncate(int64(diskSize))
		if err != nil {
			return nil, err
		}
	}
	file.Close()
	file, err = os.OpenFile(diskPath, os.O_CREATE|os.O_RDWR|os.O_SYNC, 0600)
	if err != nil {
		return nil, err
	}
	return &File{file}, nil
}

func (f *File) ReadAt(_ context.Context, p []byte, off uint64) error {
	log.Printf("FILE READ at:%d, %d bytes\n", off, len(p))
	if _, err := f.Seek(int64(off), 0); err != nil {
		return err
	}
	_, err := io.ReadFull(f, p)
	return err
}

func (f *File) WriteAt(_ context.Context, p []byte, off uint64) error {
	log.Printf("FILE WRITE at:%d, %d bytes\n", off, len(p))
	if _, err := f.Seek(int64(off), 0); err != nil {
		return err
	}
	_, err := f.Write(p)
	return err
}

func (f *File) Release() {
	f.Close()
	log.Println("FILE RELEASED")
}

func (f *File) Size(_ context.Context) (uint64, error) {
	return diskSize, nil
}
