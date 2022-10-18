package store

import "context"

type Storage interface {
	ReadAt(ctx context.Context, p []byte, off uint64) error
	WriteAt(ctx context.Context, p []byte, off uint64) error
	Size(ctx context.Context) (uint64, error)
	Release()
}
