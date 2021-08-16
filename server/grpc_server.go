package server

import (
	"fmt"
	"math"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type GRPCServer struct {
	grpcAddress string
	service     *GRPCService
	server      *grpc.Server
	listener    net.Listener
	certFile    string
	keyFile     string
	commonName  string
	logger      *logrus.Logger
}

func recoveryFunc(p interface{}) error {
	fmt.Printf("p: %+v\n", p)
	return grpc.Errorf(codes.Internal, "Unexpected error")
}

func NewGrpcServer(grpcAddress string, raftServer *RaftServer, logger *logrus.Logger) (*GRPCServer, error) {
	return NewGRPCServerWithTLS(grpcAddress, raftServer, "", "", "", logger)
}

func NewGRPCServerWithTLS(grpcAddress string, raftServer *RaftServer, certificateFile string, keyFile string, commonName string, logger *logrus.Logger) (*GRPCServer, error) {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(math.MaxInt64),
		grpc.MaxSendMsgSize(math.MaxInt64),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_recovery.StreamServerInterceptor(
					grpc_recovery.WithRecoveryHandler(recoveryFunc),
				),
				// metric.GrpcMetrics.StreamServerInterceptor(),
				grpc_logrus.StreamServerInterceptor(logrus.NewEntry(logger)),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_recovery.UnaryServerInterceptor(
					grpc_recovery.WithRecoveryHandler(recoveryFunc),
				),
				// metric.GrpcMetrics.UnaryServerInterceptor(),
				grpc_logrus.UnaryServerInterceptor(logrus.NewEntry(logger)),
			),
		),
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				//MaxConnectionIdle:     0,
				//MaxConnectionAge:      0,
				//MaxConnectionAgeGrace: 0,
				Time:    5 * time.Second,
				Timeout: 5 * time.Second,
			},
		),
	}

	if certificateFile == "" && keyFile == "" {
		logger.Info("disabling TLS")
	} else {
		logger.Info("enabling TLS")
		creds, err := credentials.NewServerTLSFromFile(certificateFile, keyFile)
		if err != nil {
			logger.Error("failed to create credentials: %s", err.Error())
		}
		opts = append(opts, grpc.Creds(creds))
	}

	server := grpc.NewServer(
		opts...,
	)

	service, err := NewGRPCService(raftServer, certificateFile, commonName, logger)
	if err != nil {
		logger.Error("failed to create key value store service: %s", err.Error())
		return nil, err
	}

	protobuf.RegisterIndexServer(server, service)
	// Initialize all metrics.
	// metric.GrpcMetrics.InitializeMetrics(server)
	// grpc_prometheus.Register(server)

	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		logger.Error("failed to create listener: %s, :%s", grpcAddress, err.Error())
		return nil, err
	}

	return &GRPCServer{
		grpcAddress: grpcAddress,
		service:     service,
		server:      server,
		listener:    listener,
		certFile:    certificateFile,
		keyFile:     keyFile,
		commonName:  commonName,
		logger:      logger,
	}, nil
}

func (s *GRPCServer) Start() error {
	if err := s.service.Start(); err != nil {
		s.logger.Errorf("GRPCServiceの起動に失敗しました: %s", err.Error())
	}
	go func() {
		_ = s.server.Serve(s.listener)
	}()
	s.logger.Infof("gRPC serverを起動しました: [address:%s]", s.grpcAddress)
	return nil
}

func (s *GRPCServer) Stop() error {
	if err := s.service.Stop(); err != nil {
		s.logger.Errorf("GRPCServiceの停止に失敗しました: %s", err.Error())
	}
	s.server.Stop()
	s.logger.Infof("gRPC serverを停止しました: [address:%s]", s.grpcAddress)
	return nil
}
