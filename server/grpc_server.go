package server

import (
	"math"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/metric/prometheus"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
	logger      logger.DefaultLogger
}

func NewGrpcServer(grpcAddress string, raftServer *RaftServer, logger logger.LogrusLogger) (*GRPCServer, error) {
	return NewGRPCServerWithTLS(grpcAddress, raftServer, "", "", "", logger)
}

func NewGRPCServerWithTLS(grpcAddress string, raftServer *RaftServer, certificateFile string, keyFile string, commonName string, logger logger.LogrusLogger) (*GRPCServer, error) {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(math.MaxInt64),
		grpc.MaxSendMsgSize(math.MaxInt64),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_recovery.StreamServerInterceptor(
					grpc_recovery.WithRecoveryHandler(recoveryFunc),
				),
				prometheus.StreamServerInterceptor(),
				grpc_validator.StreamServerInterceptor(),
				// metric.GrpcMetrics.StreamServerInterceptor(),
				grpc_logrus.StreamServerInterceptor(logrus.NewEntry(logger.Logrus())),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_recovery.UnaryServerInterceptor(
					grpc_recovery.WithRecoveryHandler(recoveryFunc),
				),
				prometheus.UnaryServerInterceptor(),
				grpc_validator.UnaryServerInterceptor(),
				// metric.GrpcMetrics.UnaryServerInterceptor(),
				grpc_logrus.UnaryServerInterceptor(logrus.NewEntry(logger.Logrus())),
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
	sericeOpts := []Option{
		WithRaftServer(raftServer),
		WithCertificate(certificateFile, commonName),
		WithLogger(logger),
	}
	service, err := NewGRPCService(sericeOpts...)
	if err != nil {
		logger.Error("failed to create key value store service: %s", err.Error())
		return nil, err
	}

	protobuf.RegisterDocsearchServer(server, service)
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
		s.logger.Errorf("GRPCServiceの停止に失敗しました[address: %s]: %s", err.Error())
	}
	s.server.Stop()
	s.logger.Infof("gRPC serverを停止しました: %s", s.grpcAddress)
	return nil
}
