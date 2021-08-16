package server

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/client"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCService struct {
	raftServer  *RaftServer
	certFile    string
	commonName  string
	logger      *logrus.Logger
	watchMutex  sync.RWMutex
	watchChans  map[chan protobuf.WatchResponse]struct{}
	peerClients map[string]*client.GRPCClient

	watchClusterStopCh chan struct{}
	watchClusterDoneCh chan struct{}

	*protobuf.UnimplementedIndexServer
}

var _ protobuf.IndexServer = (*GRPCService)(nil)

func NewGRPCService(raftServer *RaftServer, certFile, commonName string, logger *logrus.Logger) (*GRPCService, error) {
	return &GRPCService{
		raftServer:  raftServer,
		certFile:    certFile,
		commonName:  commonName,
		watchChans:  make(map[chan protobuf.WatchResponse]struct{}),
		peerClients: make(map[string]*client.GRPCClient, 0),
		logger:      logger,
	}, nil
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

	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &empty.Empty{})
		if err != nil {
			s.logger.Error(errors.ClusterInfo(err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		c := s.peerClients[clusterResp.Cluster.Leader]
		err = c.Join(req)
		if err != nil {
			s.logger.Error(errors.RequestForward(c.Target(), err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		return resp, nil
	}

	err := s.raftServer.Join(req.Id, req.Node)
	if err != nil {
		switch err {
		case errors.NodeAlreadyExists:
			s.logger.Debugf("既にノードが存在しています: %v", req)
		default:
			s.logger.Error(errors.JoinNode(err, req))
			return resp, status.Error(codes.Internal, err.Error())
		}
	}

	return resp, nil
}

func (s *GRPCService) Leave(ctx context.Context, req *protobuf.LeaveRequest) (*emptypb.Empty, error) {
	resp := &emptypb.Empty{}

	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &emptypb.Empty{})
		if err != nil {
			s.logger.Error(errors.ClusterInfo(err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		c := s.peerClients[clusterResp.Cluster.Leader]
		err = c.Leave(req)
		if err != nil {
			s.logger.Error(errors.RequestForward(c.Target(), err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		return resp, nil
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

func (s *GRPCService) Upload(ctx context.Context, req *protobuf.Documents) (*protobuf.BatchResponse, error) {
	resp := &protobuf.BatchResponse{}

	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &empty.Empty{})
		if err != nil {
			// s.logger.Error("failed to get cluster info", zap.Error(err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		c := s.peerClients[clusterResp.Cluster.Leader]
		return c.Upload(req)
	}

	if err := s.raftServer.Upload(req); err != nil {
		// s.logger.Error("failed to index documents in bulk", zap.Error(err))
		return resp, status.Error(codes.Internal, err.Error())
	}
	resp.Count = int32(len(req.Requests))
	return resp, nil
}

func (s *GRPCService) Search(ctx context.Context, req *protobuf.SearchRequest) (*protobuf.SearchResponse, error) {
	if resp, err := s.raftServer.Search(req); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (s *GRPCService) BulkDelete(ctx context.Context, req *protobuf.DeleteDocuments) (*protobuf.BatchResponse, error) {
	resp := &protobuf.BatchResponse{}

	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &empty.Empty{})
		if err != nil {
			// s.logger.Error("failed to get cluster info", zap.Error(err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		c := s.peerClients[clusterResp.Cluster.Leader]
		return c.BulkDelete(req)
	}

	if err := s.raftServer.BulkDelete(req); err != nil {
		// s.logger.Error("failed to index documents in bulk", zap.Error(err))
		return resp, status.Error(codes.Internal, err.Error())
	}
	resp.Count = int32(len(req.Requests))
	return resp, nil
}
func (s *GRPCService) Delete(ctx context.Context, in *protobuf.DeleteDocument) (*protobuf.BatchResponse, error) {
	resp := &protobuf.BatchResponse{}

	if s.raftServer.raft.State() != raft.Leader {
		clusterResp, err := s.Cluster(ctx, &empty.Empty{})
		if err != nil {
			// s.logger.Error("failed to get cluster info", zap.Error(err))
			return resp, status.Error(codes.Internal, err.Error())
		}

		c := s.peerClients[clusterResp.Cluster.Leader]
		return c.Delete(in)
	}
	req := &protobuf.DeleteDocuments{
		Requests: []*protobuf.DeleteDocument{
			in,
		},
	}
	if err := s.raftServer.BulkDelete(req); err != nil {
		// s.logger.Error("failed to index documents in bulk", zap.Error(err))
		return resp, status.Error(codes.Internal, err.Error())
	}
	resp.Count = int32(len(req.Requests))
	return resp, nil
}
func (s *GRPCService) Watch(*emptypb.Empty, protobuf.Index_WatchServer) error {
	return nil
}

func (s *GRPCService) Start() error {
	go func() {
		s.startWatchCluster(500 * time.Millisecond)
	}()

	s.logger.Info("gRPC service started")
	return nil
}

func (s *GRPCService) Stop() error {
	s.stopWatchCluster()

	s.logger.Info("gRPC service stopped")
	return nil
}

func (s *GRPCService) startWatchCluster(checkInterval time.Duration) {
	s.logger.Info("start to update cluster info")

	defer func() {
		close(s.watchClusterDoneCh)
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	timeout := 60 * time.Second
	if err := s.raftServer.WaitForDetectLeader(timeout); err != nil {
		if err == errors.TimeOut {
			// s.logger.Error("leader detection timed out", zap.Duration("timeout", timeout), zap.Error(err))
		} else {
			// s.logger.Error("failed to detect leader", zap.Error(err))
		}
	}

	for {
		select {
		case <-s.watchClusterStopCh:
			s.logger.Info("received a request to stop updating a cluster")
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

			// open clients for peer nodes
			nodes, err := s.raftServer.Nodes()
			if err != nil {
				s.logger.Warn("failed to get cluster info", err.Error())
			}
			for id, node := range nodes {
				if id == s.raftServer.nodeID {
					continue
				}

				if node.Metadata == nil || node.Metadata.GrpcAddress == "" {
					s.logger.Debug("gRPC address missing", id)
					continue
				}
				if c, ok := s.peerClients[id]; ok {
					if c.Target() != node.Metadata.GrpcAddress {
						s.logger.Debug("close client", id, c.Target())
						delete(s.peerClients, id)
						if err := c.Close(); err != nil {
							s.logger.Warn("failed to close client", id, c.Target(), err)
						}
						s.logger.Debug("create client", id, node.Metadata.GrpcAddress)
						if newClient, err := client.NewGRPCClientWithContextTLS(node.Metadata.GrpcAddress, context.TODO(), s.certFile, s.commonName); err == nil {
							s.peerClients[id] = newClient
						} else {
							s.logger.Warn("failed to create client", id, c.Target(), err)
						}
					}
				} else {
					s.logger.Debug("create client", id, node.Metadata.GrpcAddress)
					if newClient, err := client.NewGRPCClientWithContextTLS(node.Metadata.GrpcAddress, context.TODO(), s.certFile, s.commonName); err == nil {
						s.peerClients[id] = newClient
					} else {
						s.logger.Warn("failed to create client", id, c.Target(), err)
					}
				}
			}

			// close clients for non-existent peer nodes
			for id, c := range s.peerClients {
				if _, exist := nodes[id]; !exist {
					s.logger.Debug("close client", id, c.Target())
					delete(s.peerClients, id)
					if err := c.Close(); err != nil {
						s.logger.Warn("failed to close old client", id, c.Target(), err)
					}
				}
			}

			s.watchMutex.Unlock()
		}
	}
}

func (s *GRPCService) stopWatchCluster() {
	if s.watchClusterStopCh != nil {
		s.logger.Info("send a request to stop updating a cluster")
		close(s.watchClusterStopCh)
	}

	s.logger.Info("wait for the cluster watching to stop")
	<-s.watchClusterDoneCh
	s.logger.Info("the cluster watching has been stopped")

	s.logger.Info("close all peer clients")
	for id, c := range s.peerClients {
		s.logger.Debug("close client", id, c.Target())
		delete(s.peerClients, id)
		if err := c.Close(); err != nil {
			s.logger.Warn("failed to close client", id, c.Target(), err)
		}
	}
}

func (s *GRPCService) Insert(ctx context.Context, req *protobuf.Document) (*emptypb.Empty, error) {
	batchReq := &protobuf.Documents{
		Requests: []*protobuf.Document{
			req,
		},
	}
	if _, err := s.Upload(ctx, batchReq); err != nil {
		s.logger.Error(err)
		return nil, err
	} else {
		return &emptypb.Empty{}, nil
	}
}
