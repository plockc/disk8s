package replica

import (
	"context"
	"fmt"
	"net"

	"github.com/plockc/disk8s/nbd/internal/store"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type dataDiskServer struct {
	store.Storage
	UnimplementedDataDiskServer
}

type Server interface {
	HandleRequests(ctx context.Context) error
}

func NewDataDiskServer(storage store.Storage) Server {
	return &dataDiskServer{
		Storage: storage,
	}
}

func (s dataDiskServer) Read(ctx context.Context, req *ReadReq) (*ReadResp, error) {
	buff := make([]byte, req.Size)
	err := s.Storage.ReadAt(buff, uint64(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ReadResp{
		Data: buff,
	}, nil
}

func (s dataDiskServer) Write(ctx context.Context, req *WriteReq) (*WriteResp, error) {
	err := s.Storage.WriteAt(req.Data, uint64(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &WriteResp{}, nil
}

func (s *dataDiskServer) HandleRequests(ctx context.Context) error {
	listener, err := net.Listen("tcp", "localhost:10808")
	if err != nil {
		return fmt.Errorf("failed to listen on port 10808: %w", err)
	}
	srvr := grpc.NewServer()
	RegisterDataDiskServer(srvr, s)
	// if parent context stops we can gracefully stop the server
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		srvr.GracefulStop()
	}()
	err = srvr.Serve(listener)
	return err
}
