package client

import (
	"context"
	"math"
	"time"

	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCClient struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *grpc.ClientConn
	client protobuf.IndexClient
	logger *logrus.Logger
}

func NewGRPCClient(grpcAddress string) (*GRPCClient, error) {
	return NewGRPCClientWithContext(grpcAddress, context.Background())
}

func NewGRPCClientWithContext(grpcAddress string, baseCtx context.Context) (*GRPCClient, error) {
	return NewGRPCClientWithContextTLS(grpcAddress, baseCtx, "", "")
}

func NewGRPCClientWithContextTLS(grpcAddress string, baseCtx context.Context, certFile, commonName string) (*GRPCClient, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(math.MaxInt32),
			grpc.MaxCallRecvMsgSize(math.MaxInt32),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                1 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	ctx, cancel := context.WithCancel(baseCtx)
	if certFile == "" {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(certFile, commonName)
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.DialContext(ctx, grpcAddress, dialOpts...)
	if err != nil {
		cancel()
		return nil, err
	}
	return &GRPCClient{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
		client: protobuf.NewIndexClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	c.cancel()
	if c.conn != nil {
		return c.conn.Close()
	}
	return c.ctx.Err()
}

func (c *GRPCClient) Target() string {
	return c.conn.Target()
}

func (c *GRPCClient) Join(req *protobuf.JoinRequest, opts ...grpc.CallOption) error {
	if _, err := c.client.Join(c.ctx, req, opts...); err != nil {
		return err
	}
	return nil
}

func (c *GRPCClient) Leave(req *protobuf.LeaveRequest, opts ...grpc.CallOption) error {
	if _, err := c.client.Leave(c.ctx, req, opts...); err != nil {
		return err
	}
	return nil
}

func (c *GRPCClient) Node(opts ...grpc.CallOption) (*protobuf.NodeResponse, error) {
	if resp, err := c.client.Node(c.ctx, &emptypb.Empty{}, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (c *GRPCClient) Cluster(opts ...grpc.CallOption) (*protobuf.ClusterResponse, error) {
	if resp, err := c.client.Cluster(c.ctx, &emptypb.Empty{}, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (c *GRPCClient) Upload(req *protobuf.Documents, opts ...grpc.CallOption) (*protobuf.BatchResponse, error) {
	if resp, err := c.client.Upload(c.ctx, req, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}
func (c *GRPCClient) BulkDelete(req *protobuf.DeleteDocuments, opts ...grpc.CallOption) (*protobuf.BatchResponse, error) {
	if resp, err := c.client.BulkDelete(c.ctx, req, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}

}
func (c *GRPCClient) Delete(req *protobuf.DeleteDocument, opts ...grpc.CallOption) (*protobuf.BatchResponse, error) {
	if resp, err := c.client.Delete(c.ctx, req, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}

}
func (c *GRPCClient) Search(req *protobuf.SearchRequest, opts ...grpc.CallOption) (*protobuf.SearchResponse, error) {
	if resp, err := c.client.Search(c.ctx, req, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}

}
func (c *GRPCClient) Watch(opts ...grpc.CallOption) (protobuf.Index_WatchClient, error) {
	if resp, err := c.client.Watch(c.ctx, &emptypb.Empty{}, opts...); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}
