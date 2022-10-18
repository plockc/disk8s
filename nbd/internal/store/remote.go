package store

import (
	"context"
	"log"

	"github.com/plockc/disk8s/nbd/replica/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Remote struct {
	client pb.DataDiskClient
	conn   *grpc.ClientConn
}

func NewRemote(hostPort string) (Storage, error) {
	conn, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Remote{
		client: pb.NewDataDiskClient(conn),
		conn:   conn,
	}, nil
}

func (r *Remote) ReadAt(ctx context.Context, p []byte, off uint64) error {
	log.Printf("GRPC REQUEST READ at:%d, %d bytes\n", off, len(p))
	resp, err := r.client.Read(ctx, &pb.ReadReq{Size: uint32(len(p)), Offset: off})
	if err != nil {
		return err
	}
	copy(p, resp.Data)
	return nil
}

func (r *Remote) WriteAt(ctx context.Context, p []byte, off uint64) error {
	log.Printf("GRPC REQUEST WRITE at:%d, %d bytes\n", off, len(p))
	_, err := r.client.Write(ctx, &pb.WriteReq{Data: p, Offset: off})
	return err
}

func (r *Remote) Release() {
	log.Println("GRPC CLOSED")
	r.conn.Close()
}

func (r *Remote) Size(ctx context.Context) (uint64, error) {
	log.Printf("GRPC REQUEST SIZE")
	resp, err := r.client.Size(ctx, &pb.SizeReq{})
	if err != nil {
		return 0, err
	}
	return resp.Size, nil
}
