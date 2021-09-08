package server

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/client"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/metric/prometheus"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type localLogger struct {
	*logrus.Logger
}

func (l *localLogger) Logrus() *logrus.Logger {
	return l.Logger
}

func (l *localLogger) Write(p []byte) (int, error) {
	l.Info(string(p))
	return len(p), nil
}

type Option func(service *GRPCService)

func WithRaftServer(raftServer *RaftServer) Option {
	return func(service *GRPCService) {
		service.raftServer = raftServer
	}
}

func WithCertificate(certFile, commonName string) Option {
	return func(service *GRPCService) {
		service.certFile = certFile
		service.commonName = commonName
	}
}

func WithLogger(logger logger.LogrusLogger) Option {
	return func(service *GRPCService) {
		service.logger = logger
	}
}

type GRPCService struct {
	raftServer  *RaftServer
	certFile    string
	commonName  string
	logger      logger.LogrusLogger
	watchMutex  sync.RWMutex
	watchChans  map[chan protobuf.WatchResponse]struct{}
	peerClients map[string]*client.GRPCClient

	watchClusterStopCh chan struct{}
	watchClusterDoneCh chan struct{}

	*protobuf.UnimplementedDocsearchServer
}

var _ protobuf.DocsearchServer = (*GRPCService)(nil)

func NewGRPCService(opts ...Option) (*GRPCService, error) {
	service := &GRPCService{
		logger: &localLogger{
			Logger: logrus.New(),
		},
		watchChans:         make(map[chan protobuf.WatchResponse]struct{}),
		peerClients:        make(map[string]*client.GRPCClient, 0),
		watchClusterStopCh: make(chan struct{}),
		watchClusterDoneCh: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(service)
	}
	return service, nil
}

func (s *GRPCService) LivenessCheck(ctx context.Context, in *emptypb.Empty) (*protobuf.LivenessCheckResponse, error) {
	resp := &protobuf.LivenessCheckResponse{}
	resp.Alive = true
	return resp, nil
}

func (s *GRPCService) ReadinessCheck(context.Context, *emptypb.Empty) (*protobuf.ReadinessCheckResponse, error) {
	resp := &protobuf.ReadinessCheckResponse{}

	timeout := 10 * time.Second
	if err := s.raftServer.WaitForDetectLeader(timeout); err != nil {
		s.logger.Errorf("リーダーノードが見つかりません: %s", err.Error())
		return resp, status.Error(codes.Internal, err.Error())
	}

	if s.raftServer.State() == raft.Candidate || s.raftServer.State() == raft.Shutdown {
		err := errors.NodeAlreadyExists
		s.logger.Error(err.Error())
		return resp, status.Error(codes.Internal, err.Error())
	}

	resp.Ready = true

	return resp, nil
}

func (s *GRPCService) Node(ctx context.Context, req *empty.Empty) (*protobuf.NodeResponse, error) {
	resp := &protobuf.NodeResponse{}

	node, err := s.raftServer.Node()
	if err != nil {
		s.logger.Errorf("ノード情報の取得に失敗しました: %s", err.Error())
		return resp, status.Error(codes.Internal, err.Error())
	}

	resp.Node = node

	return resp, nil
}

func (s *GRPCService) Join(ctx context.Context, req *protobuf.JoinRequest) (*emptypb.Empty, error) {
	resp := &empty.Empty{}
	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			if err := c.Join(req); err != nil {
				s.logger.Error(errors.RequestForward(c.Target(), err))
				return resp, status.Error(codes.Internal, err.Error())
			}
			return resp, nil
		}
	}

	err := s.raftServer.Join(req.Id, req.Node)
	if err != nil {
		fields := logrus.Fields{"id": req.Id, "address": req.Node.RaftAddress}
		switch err {
		case errors.NodeAlreadyExists:
			s.logger.Logrus().WithFields(fields).Debugf("既にノードが存在しています")
			return resp, nil
		default:
			s.logger.Logrus().WithFields(fields).Error(errors.JoinNode(err, req))
			return resp, status.Error(codes.Internal, err.Error())
		}
	}

	return resp, nil
}

func (s *GRPCService) Leave(ctx context.Context, req *protobuf.LeaveRequest) (*emptypb.Empty, error) {
	resp := &emptypb.Empty{}

	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			if err := c.Leave(req); err != nil {
				s.logger.Error(errors.RequestForward(c.Target(), err))
				return resp, status.Error(codes.Internal, err.Error())
			}
			return resp, nil
		}
	}

	err := s.raftServer.Leave(req.Id)
	if err != nil {
		s.logger.Error(errors.NodeLeave(err, req))
		return resp, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func (s *GRPCService) Cluster(ctx context.Context, req *empty.Empty) (*protobuf.ClusterResponse, error) {
	resp := &protobuf.ClusterResponse{}

	cluster := &protobuf.Cluster{}

	nodes, err := s.raftServer.Nodes()
	if err != nil {
		s.logger.Error(errors.ClusterInfo(err))
		return resp, status.Error(codes.Internal, err.Error())
	}

	for id, node := range nodes {
		if id == s.raftServer.nodeID {
			node.State = s.raftServer.StateStr()
		} else {
			c := s.peerClients[id]
			nodeResp, err := c.Node()
			if err != nil {
				node.State = raft.Shutdown.String()
				s.logger.Error(errors.NodeInfo(err))
			} else {
				node.State = nodeResp.Node.State
			}
		}
	}
	cluster.Nodes = nodes

	serverID, err := s.raftServer.LeaderID(60 * time.Second)
	if err != nil {
		s.logger.Error(errors.ClusterInfo(err))
		return resp, status.Error(codes.Internal, err.Error())
	}
	cluster.Leader = string(serverID)

	resp.Cluster = cluster

	return resp, nil
}

func (s *GRPCService) Upload(ctx context.Context, req *protobuf.Documents) (*emptypb.Empty, error) {
	resp := &emptypb.Empty{}
	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			return c.Upload(req)
		}
	}
	if err := s.raftServer.Upload(req); err != nil {
		return resp, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func (s *GRPCService) Delete(ctx context.Context, in *protobuf.DeleteDocument) (*emptypb.Empty, error) {
	resp := &emptypb.Empty{}
	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			return c.Delete(in)
		}
	}
	req := &protobuf.DeleteDocuments{
		Requests: []*protobuf.DeleteDocument{
			in,
		},
	}
	if err := s.raftServer.BulkDelete(req); err != nil {
		return resp, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func (s *GRPCService) BulkDelete(ctx context.Context, req *protobuf.DeleteDocuments) (*emptypb.Empty, error) {
	resp := &emptypb.Empty{}
	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			return c.BulkDelete(req)
		}
	}
	if err := s.raftServer.BulkDelete(req); err != nil {
		return resp, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func (s *GRPCService) Search(ctx context.Context, req *protobuf.SearchRequest) (*protobuf.SearchResponse, error) {
	if resp, err := s.raftServer.Search(req); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (s *GRPCService) UploadDictionary(ctx context.Context, req *protobuf.UserDictionaryRecords) (*protobuf.DictionaryResponse, error) {
	resp := &protobuf.DictionaryResponse{
		Results: false,
	}
	if c, err := s.getLeaderCluster(ctx); err != nil {
		return resp, err
	} else {
		if c != nil {
			return c.UploadDictionary(req)
		}
	}
	var err error
	if resp, err = s.raftServer.AddUserDictionary(req); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (s *GRPCService) DeleteDictionary(ctx context.Context, req *protobuf.DeleteDictionaryRequest) (*protobuf.DictionaryResponse, error) {
	if c, err := s.getLeaderCluster(ctx); err != nil {
		resp := &protobuf.DictionaryResponse{
			Results: false,
		}
		return resp, err
	} else {
		if c != nil {
			return c.DeleteDictionary(req)
		}
	}
	if resp, err := s.raftServer.RemoveDictionary(req); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (s *GRPCService) Watch(req *empty.Empty, server protobuf.Docsearch_WatchServer) error {
	chans := make(chan protobuf.WatchResponse)

	s.watchMutex.Lock()
	s.watchChans[chans] = struct{}{}
	s.watchMutex.Unlock()

	defer func() {
		s.watchMutex.Lock()
		delete(s.watchChans, chans)
		s.watchMutex.Unlock()
		close(chans)
	}()

	for resp := range chans {
		if err := server.Send(&resp); err != nil {
			s.logger.Error("failed to send watch data: %s, err: %s", resp.Event, err.Error())
			return status.Error(codes.Internal, err.Error())
		}
	}

	return nil
}

func (s *GRPCService) Metrics(ctx context.Context, req *empty.Empty) (*protobuf.MetricsResponse, error) {
	resp := &protobuf.MetricsResponse{}

	var err error

	gather, err := prometheus.Registry.Gather()
	if err != nil {
		s.logger.Errorf("メトリクスの収集に失敗しました: %s", err)
	}
	out := &bytes.Buffer{}
	for _, mf := range gather {
		if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
			s.logger.Errorf("metric familyの解析に失敗しました: %s", err)
		}
	}

	resp.Metrics = out.Bytes()

	return resp, nil
}

func (s *GRPCService) Start() error {
	go func() {
		s.startWatchCluster(500 * time.Millisecond)
	}()

	s.logger.Info("gRPC service を開始しました")
	return nil
}

func (s *GRPCService) Stop() error {
	s.stopWatchCluster()

	s.logger.Info("gRPC service を停止しました")
	return nil
}

func (s *GRPCService) startWatchCluster(checkInterval time.Duration) {
	s.logger.Info("クラスター情報の更新を開始します")

	defer func() {
		close(s.watchClusterDoneCh)
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	timeout := 60 * time.Second
	if err := s.raftServer.WaitForDetectLeader(timeout); err != nil {
		if err == errors.TimeOut {
			s.logger.Errorf("リーダーの検出にタイムアウトしました")
		} else {
			s.logger.Errorf("リーダーの検出に失敗しました: %s", err.Error())
		}
	}

	for {
		select {
		case <-s.watchClusterStopCh:
			s.logger.Info("クラスターの更新を停止する要求を受け取りました")
			return
		case event := <-s.raftServer.applyCh:
			watchResp := &protobuf.WatchResponse{
				Event: event,
			}
			for c := range s.watchChans {
				c <- *watchResp
			}
		case <-ticker.C:
			s.watchMutex.Lock()

			nodes, err := s.raftServer.Nodes()
			if err != nil {
				s.logger.Warnf("クラスター情報の取得に失敗しました: %s", err.Error())
			}
			for id, node := range nodes {
				if id == s.raftServer.nodeID {
					continue
				}

				if node.Metadata == nil || node.Metadata.GrpcAddress == "" {
					s.logger.Debugf("gRPCアドレスがありません: %s", id)
					continue
				}
				if c, ok := s.peerClients[id]; ok {
					if c.Target() != node.Metadata.GrpcAddress {
						s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Debug("gRPC clientを終了します")
						delete(s.peerClients, id)
						if err := c.Close(); err != nil {
							s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Warnf("gRPC clientを終了することができませんでした: %s", err)
						}
						s.logger.Logrus().WithFields(logrus.Fields{"id": id}).Debugf("create gRPC client: %s", node.Metadata.GrpcAddress)
						if newClient, err := client.NewGRPCClientWithContextTLS(node.Metadata.GrpcAddress, context.TODO(), s.certFile, s.commonName); err == nil {
							s.peerClients[id] = newClient
						} else {
							s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Warnf("failed to create gRPC client: %s", err)
						}
					}
				} else {
					s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": node.Metadata.GrpcAddress}).Debugf("create gRPC client")
					if newClient, err := client.NewGRPCClientWithContextTLS(node.Metadata.GrpcAddress, context.TODO(), s.certFile, s.commonName); err == nil {
						s.peerClients[id] = newClient
					} else {
						s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Warnf("failed to create gRPC client: %s", err)
					}
				}
			}

			// close clients for non-existent peer nodes
			for id, c := range s.peerClients {
				if _, exist := nodes[id]; !exist {
					s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Debugf("close client")
					delete(s.peerClients, id)
					if err := c.Close(); err != nil {
						s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Warnf("failed to close old client: %s", err)
					}
				}
			}

			s.watchMutex.Unlock()
		}
	}
}

func (s *GRPCService) stopWatchCluster() {
	if s.watchClusterStopCh != nil {
		s.logger.Info("クラスターの更新を停止するリクエストを送信します")
		close(s.watchClusterStopCh)
	}

	s.logger.Info("クラスターの監視が停止するのを待っています")
	<-s.watchClusterDoneCh
	s.logger.Info("クラスターの監視が停止されました")

	s.logger.Info("すべての接続されているgRPC clientを停止します")
	for id, c := range s.peerClients {
		s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Debug("close client")
		delete(s.peerClients, id)
		if err := c.Close(); err != nil {
			s.logger.Logrus().WithFields(logrus.Fields{"id": id, "address": c.Target()}).Warnf("gRPC clientの停止が失敗しました: %s", err)
		}
	}
}

func (s *GRPCService) getLeaderCluster(ctx context.Context) (*client.GRPCClient, error) {
	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &empty.Empty{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		c := s.peerClients[clusterResp.Cluster.Leader]
		return c, nil
	}
	return nil, nil
}

// func getTenantAndIndex(md metadata.MD) (tenant, index string) {
// 	f := func(key string) string {
// 		values := md.Get(key)
// 		if len(values) > 0 {
// 			return values[0]
// 		}
// 		return ""
// 	}
// 	tenant = f(XTenantID)
// 	index = f(XIndexKey)
// 	return
// }
