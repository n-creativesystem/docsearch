package server

import (
	"context"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-creativesystem/docsearch/marshaler"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/proto"
)

func responseFilter(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	switch resp.(type) {
	default:
		w.Header().Set("Content-Type", marshaler.DefaultContentType)
	}

	return nil
}

type GRPCGateway struct {
	httpAddress string
	grpcAddress string

	cancel   context.CancelFunc
	listener net.Listener
	mux      *runtime.ServeMux

	certFile string
	keyFile  string

	logger *logrus.Logger
}

func NewGRPCGateway(httpAddress string, grpcAddress string, certFile string, keyFile string, commonName string, logger *logrus.Logger) (*GRPCGateway, error) {
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

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, new(marshaler.GatewayMarshaler)),
		runtime.WithForwardResponseOption(responseFilter),
	)

	if certFile == "" {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(certFile, commonName)
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	err := protobuf.RegisterIndexHandlerFromEndpoint(ctx, mux, grpcAddress, dialOpts)
	if err != nil {
		logger.Error("failed to register KVS handler from endpoint", err)
		return nil, err
	}

	listener, err := net.Listen("tcp", httpAddress)
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

	s.logger.Info("gRPC gateway started", s.httpAddress)
	return nil
}

func (s *GRPCGateway) Stop() error {
	defer s.cancel()

	err := s.listener.Close()
	if err != nil {
		s.logger.Error("failed to close listener", s.listener.Addr().String(), err)
	}

	s.logger.Info("gRPC gateway stopped", s.httpAddress)
	return nil
}
