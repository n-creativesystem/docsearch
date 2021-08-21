package server

import (
	"context"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/marshaler"
	"github.com/n-creativesystem/docsearch/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func responseFilter(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	switch resp.(type) {
	default:
		w.Header().Set("Content-Type", marshaler.DefaultContentType)
	}

	return nil
}

const (
	XTenantID  = "X-Tenant-ID"
	XIndexKey  = "X-Index-Key"
	XRequestID = "X-Request-ID"
)

func withMetadata(ctx context.Context, req *http.Request) metadata.MD {
	f := func(key, default_ string) string {
		if v := req.Header.Get(key); v != "" {
			return v
		}
		return default_
	}
	return metadata.New(map[string]string{
		XTenantID:  f(XTenantID, "default"),
		XIndexKey:  f(XIndexKey, "index"),
		XRequestID: req.Header.Get(XRequestID),
	})
}

type GRPCGateway struct {
	httpAddress string
	grpcAddress string

	cancel   context.CancelFunc
	listener net.Listener
	mux      *runtime.ServeMux

	certFile string
	keyFile  string

	logger logger.DefaultLogger
}

func NewGRPCGateway(httpAddress string, grpcAddress string, certFile string, keyFile string, commonName string, logger logger.DefaultLogger) (*GRPCGateway, error) {
	var err error
	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(math.MaxInt64),
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
		),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                1 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			},
		),
	}

	baseCtx := context.TODO()
	ctx, cancel := context.WithCancel(baseCtx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	mux := runtime.NewServeMux(
		runtime.WithMetadata(withMetadata),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, new(marshaler.GatewayMarshaler)),
		runtime.WithForwardResponseOption(responseFilter),
	)

	if certFile == "" {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		var creds credentials.TransportCredentials
		creds, err = credentials.NewClientTLSFromFile(certFile, commonName)
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	err = protobuf.RegisterDocsearchHandlerFromEndpoint(ctx, mux, grpcAddress, dialOpts)
	if err != nil {
		logger.Error("failed to register KVS handler from endpoint", err)
		return nil, err
	}
	var listener net.Listener
	listener, err = net.Listen("tcp", httpAddress)
	if err != nil {
		logger.Error("failed to create index service", err)
		return nil, err
	}

	return &GRPCGateway{
		httpAddress: httpAddress,
		grpcAddress: grpcAddress,
		listener:    listener,
		mux:         mux,
		cancel:      cancel,
		certFile:    certFile,
		keyFile:     keyFile,
		logger:      logger,
	}, nil
}

func (s *GRPCGateway) Start() error {
	if s.certFile == "" && s.keyFile == "" {
		go func() {
			_ = http.Serve(s.listener, s.mux)
		}()
	} else {
		go func() {
			_ = http.ServeTLS(s.listener, s.mux, s.certFile, s.keyFile)
		}()
	}

	s.logger.Infof("gRPC gatewayを開始しました: %s", s.httpAddress)
	return nil
}

func (s *GRPCGateway) Stop() error {
	defer s.cancel()

	err := s.listener.Close()
	if err != nil {
		s.logger.Errorf("gRPCリスナーのクローズに失敗しました[address: %s]: %s", s.listener.Addr().String(), err)
		return err
	}

	s.logger.Infof("gRPC gatewayを停止しました: %s", s.httpAddress)
	return nil
}
