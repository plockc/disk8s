package replica

import (
	"context"

	"github.com/plockc/disk8s/nbd/internal/grpc"
	"github.com/plockc/disk8s/nbd/internal/store"
)

type DataDiskServer struct {
	store.Storage
}

func (s *DataDiskServer) Read(ctx context.Context, req *grpc.ReadReq) (*grpc.ReadResp, error) {
	buff := make([]byte, req.Size)
	err := s.Storage.ReadAt(buff, uint64(req.Offset))
	resp := &grpc.ReadResp{
		Data: buff,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

func HandleRequests(ctx context.Context) {

}
