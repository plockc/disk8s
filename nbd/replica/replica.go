package replica

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/plockc/disk8s/nbd/internal/store"
	"github.com/plockc/disk8s/nbd/replica/pb"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type dataDiskServer struct {
	store.Storage
	pb.UnimplementedDataDiskServer
}

type Server interface {
	HandleRequests(ctx context.Context) error
}

func NewDataDiskServer(storage store.Storage) Server {
	return &dataDiskServer{
		Storage: storage,
	}
}

func (s dataDiskServer) Read(ctx context.Context, req *pb.ReadReq) (*pb.ReadResp, error) {
	log.Printf("GRPC RESPOND READ at:%d, %d bytes\n", req.Offset, req.Size)
	buff := make([]byte, req.Size)
	err := s.Storage.ReadAt(ctx, buff, uint64(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ReadResp{
		Data: buff,
	}, nil
}

func (s dataDiskServer) Write(ctx context.Context, req *pb.WriteReq) (*pb.WriteResp, error) {
	log.Printf("GRPC RESPOND WRITE at:%d, %d bytes\n", req.Offset, len(req.Data))
	err := s.Storage.WriteAt(ctx, req.Data, uint64(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.WriteResp{}, nil
}

func (s dataDiskServer) Size(ctx context.Context, req *pb.SizeReq) (*pb.SizeResp, error) {
	log.Printf("GRPC RESPOND SIZE")
	size, err := s.Storage.Size(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SizeResp{Size: size}, nil
}

func (s *dataDiskServer) HandleRequests(ctx context.Context) error {
	listener, err := net.Listen("tcp", ":10808")
	if err != nil {
		return fmt.Errorf("failed to listen on port 10808: %w", err)
	}
	srvr := grpc.NewServer()
	pb.RegisterDataDiskServer(srvr, s)
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
