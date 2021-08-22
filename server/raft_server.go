package server

import (
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	raftbadgerdb "github.com/BBVA/raft-badger"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/config"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/fsm"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/metric/prometheus"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage/badgerdb"
	"github.com/n-creativesystem/docsearch/utils"
	"github.com/n-creativesystem/docsearch/utils/adapter"
	"google.golang.org/protobuf/proto"
)

type RaftServer struct {
	nodeID        string
	raftAddress   string
	dataDirectory string
	bootstrap     bool
	raft          *raft.Raft
	fsm           *fsm.RaftFSM
	transport     *raft.NetworkTransport
	logger        logger.LogrusLogger
	raftStore     *raftbadgerdb.BadgerStore

	watchClusterStopCh chan struct{}
	watchClusterDoneCh chan struct{}

	applyCh chan *protobuf.Event
}

func NewRaftServer(conf *config.AppConfig, raftAddress string, bootstrap bool, log logger.LogrusLogger) (*RaftServer, error) {
	fsm, err := fsm.NewRaftFSM(conf, log)
	if err != nil {
		log.Errorf("RaftFSMのインスタンス生成に失敗しました: %s", err.Error())
		return nil, err
	}
	return &RaftServer{
		nodeID:             conf.GetNodeID(),
		raftAddress:        raftAddress,
		dataDirectory:      conf.GetDocSearchIndexDirectory(),
		bootstrap:          bootstrap,
		logger:             log,
		fsm:                fsm,
		watchClusterStopCh: make(chan struct{}),
		watchClusterDoneCh: make(chan struct{}),

		applyCh: make(chan *protobuf.Event, 1024),
	}, nil
}

// Start is raft server の開始
func (s *RaftServer) Start() error {
	if err := s.fsm.Start(); err != nil {
		return err
	}
	config := raft.DefaultConfig()
	config.Logger = adapter.NewHcLogAdapter(s.logger, "docsearch")
	config.LocalID = raft.ServerID(s.nodeID)
	config.SnapshotThreshold = 1024
	addr, err := net.ResolveTCPAddr("tcp", s.raftAddress)
	if err != nil {
		s.logger.Errorf("resolver tcp address error: %s", err.Error())
		return err
	}
	s.transport, err = raft.NewTCPTransport(s.raftAddress, addr, 3, 10*time.Second, io.Discard)
	if err != nil {
		s.logger.Errorf("TCP Transportのインスタンス生成に失敗しました: %s", err.Error())
		return err
	}
	snapshotStore, err := raft.NewFileSnapshotStore(s.dataDirectory, 2, io.Discard)
	if err != nil {
		s.logger.Errorf("スナップショットストアのインスタンス生成に失敗しました: %s", err.Error())
		return err
	}
	raftStorePath := filepath.Join(s.dataDirectory, "raft")
	s.raftStore, err = badgerdb.New(raftStorePath)
	if err != nil {
		s.logger.Errorf("log store badgerDB error: %s", err.Error())
		return err
	}
	s.raft, err = raft.NewRaft(config, s.fsm, s.raftStore, s.raftStore, snapshotStore, s.transport)
	if err != nil {
		s.logger.Errorf("Raftインスタンスの生成に失敗しました: %s", err.Error())
		return err
	}
	if s.bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: s.transport.LocalAddr(),
				},
			},
		}
		s.raft.BootstrapCluster(configuration)
	}
	go func() {
		s.startWatchCluster(500 * time.Millisecond)
	}()
	s.logger.Infof("Raft serverを起動しました: %s", s.raftAddress)
	return nil
}

// LeaderAddress is リーダーアドレスの取得
func (s *RaftServer) LeaderAddress(timeout time.Duration) (raft.ServerAddress, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			leaderAddr := s.raft.Leader()
			if leaderAddr != "" {
				s.logger.Infof("リーダーのアドレスが検出されました: %s", leaderAddr)
				return leaderAddr, nil
			}
		case <-timer.C:
			err := errors.TimeOut
			s.logger.Errorf("リーダーのアドレス検出エラー: %s", err.Error())
			return "", err
		}
	}
}

// LeaderID is リーダーIDの取得
func (s *RaftServer) LeaderID(timeout time.Duration) (raft.ServerID, error) {
	leaderAddr, err := s.LeaderAddress(timeout)
	if err != nil {
		s.logger.Errorf("Leader id error: %s", err.Error())
		return "", err
	}
	s.logger.Infof("leader address: %s", leaderAddr)
	cf := s.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		s.logger.Errorf("Leader id error: %s", err.Error())
		return "", err
	}
	for _, server := range cf.Configuration().Servers {
		s.logger.Infof("server.Address: %s", server.Address)
		if server.Address == leaderAddr {
			s.logger.Infof("detected a leader ID: %s", server.ID)
			return server.ID, nil
		}
	}
	err = errors.NotFoundLeader
	s.logger.Errorf("Leader id error: %s", err.Error())
	return "", err
}

// Stop is raft server の停止
func (s *RaftServer) Stop() error {
	s.applyCh <- nil
	s.logger.Info("アプライチャンネルが閉じられました")

	s.stopWatchCluster()

	if err := s.fsm.Close(); err != nil {
		s.logger.Error("FSMが閉じれませんでした: %s", err.Error())
	}
	s.logger.Info("Raft FSMを閉じています")
	if future := s.raft.Shutdown(); future.Error() != nil {
		err := future.Error()
		s.logger.Errorf("Raftのシャットダウンに失敗しました: %s", err.Error())
	}

	if s.raftStore != nil {
		s.raftStore.Close()
	}
	s.logger.Info("Raftがシャットダウンされました: %s", s.raftAddress)
	return nil
}

// State is raft serverの状態を取得
func (s *RaftServer) State() raft.RaftState {
	return s.raft.State()
}

// StateStr is raft serverの状態を文字列で取得
func (s *RaftServer) StateStr() string {
	return s.State().String()
}

// Exist is ノードが既にクラスター内に存在しているか
func (s *RaftServer) Exist(id string) (bool, error) {
	exist := false
	cf := s.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		s.logger.Error(errors.RaftConfigration(err))
		return false, err
	}
	for _, server := range cf.Configuration().Servers {
		if server.ID == raft.ServerID(id) {
			s.logger.Debugf("ノートは既にクラスターに参加しています: %s", id)
			exist = true
			break
		}
	}
	return exist, nil
}

// WaitForDetectLeader is リーダーの検出待ち
func (s *RaftServer) WaitForDetectLeader(timeout time.Duration) error {
	if _, err := s.LeaderAddress(timeout); err != nil {
		s.logger.Error("リーダーの検出を待つことができませんでした: %s", err.Error())
		return err
	}

	return nil
}

// startWatchCluster is クラスタの状態監視を開始
func (s *RaftServer) startWatchCluster(checkInterval time.Duration) {
	s.logger.Info("クラスター情報の更新を開始します")

	defer func() {
		close(s.watchClusterDoneCh)
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	timeout := 60 * time.Second
	if err := s.WaitForDetectLeader(timeout); err != nil {
		if err == errors.TimeOut {
			s.logger.Error("リーダーの検出がタイムアウトしました", timeout, err)
		} else {
			s.logger.Error("リーダーの検出に失敗しました", err)
		}
	}

	for {
		select {
		case <-s.watchClusterStopCh:
			s.logger.Info("クラスター更新の停止要求を受け取りました")
			return
		case <-s.raft.LeaderCh():
			s.logger.Infof("リーダーに選出されました: %s", string(s.raft.Leader()))
		case event := <-s.fsm.ApplyCh:
			s.applyCh <- event
		case <-ticker.C:
			raftStats := s.raft.Stats()
			s.logger.Debugf("%v", raftStats)
			state, ok := raftStats["state"]
			if ok {
				stateMetrics := prometheus.RaftStateMetric.WithLabelValues(s.nodeID)
				switch strings.ToLower(state) {
				case "follower":
					stateMetrics.Set(float64(raft.Follower))
				case "candidate":
					stateMetrics.Set(float64(raft.Candidate))
				case "leader":
					stateMetrics.Set(float64(raft.Leader))
				case "shutdown":
					stateMetrics.Set(float64(raft.Shutdown))
				}
			}
			if term, err := strconv.ParseFloat(raftStats["term"], 64); err == nil {
				prometheus.RaftTermMetric.WithLabelValues(s.nodeID).Set(term)
			}

			if lastLogIndex, err := strconv.ParseFloat(raftStats["last_log_index"], 64); err == nil {
				prometheus.RaftLastLogIndexMetric.WithLabelValues(s.nodeID).Set(lastLogIndex)
			}

			if lastLogTerm, err := strconv.ParseFloat(raftStats["last_log_term"], 64); err == nil {
				prometheus.RaftLastLogTermMetric.WithLabelValues(s.nodeID).Set(lastLogTerm)
			}

			if commitIndex, err := strconv.ParseFloat(raftStats["commit_index"], 64); err == nil {
				prometheus.RaftCommitIndexMetric.WithLabelValues(s.nodeID).Set(commitIndex)
			}

			if appliedIndex, err := strconv.ParseFloat(raftStats["applied_index"], 64); err == nil {
				prometheus.RaftAppliedIndexMetric.WithLabelValues(s.nodeID).Set(appliedIndex)
			}

			if fsmPending, err := strconv.ParseFloat(raftStats["fsm_pending"], 64); err == nil {
				prometheus.RaftFsmPendingMetric.WithLabelValues(s.nodeID).Set(fsmPending)
			}

			if lastSnapshotIndex, err := strconv.ParseFloat(raftStats["last_snapshot_index"], 64); err == nil {
				prometheus.RaftLastSnapshotIndexMetric.WithLabelValues(s.nodeID).Set(lastSnapshotIndex)
			}

			if lastSnapshotTerm, err := strconv.ParseFloat(raftStats["last_snapshot_term"], 64); err == nil {
				prometheus.RaftLastSnapshotTermMetric.WithLabelValues(s.nodeID).Set(lastSnapshotTerm)
			}

			if latestConfigurationIndex, err := strconv.ParseFloat(raftStats["latest_configuration_index"], 64); err == nil {
				prometheus.RaftLatestConfigurationIndexMetric.WithLabelValues(s.nodeID).Set(latestConfigurationIndex)
			}

			if numPeers, err := strconv.ParseFloat(raftStats["num_peers"], 64); err == nil {
				prometheus.RaftNumPeersMetric.WithLabelValues(s.nodeID).Set(numPeers)
			}

			if lastContact, err := strconv.ParseFloat(raftStats["last_contact"], 64); err == nil {
				prometheus.RaftLastContactMetric.WithLabelValues(s.nodeID).Set(lastContact)
			}

			if nodes, err := s.Nodes(); err == nil {
				prometheus.RaftNumNodesMetric.WithLabelValues(s.nodeID).Set(float64(len(nodes)))
			}

		}
	}
}

// stopWatchCluster is クラスタの状態監視を停止
func (s *RaftServer) stopWatchCluster() {
	if s.watchClusterStopCh != nil {
		s.logger.Info("クラスタの更新を停止するリクエストを送信します")
		close(s.watchClusterStopCh)
	}

	s.logger.Info("クラスターの更新が停止しているのを待っています")
	<-s.watchClusterDoneCh
	s.logger.Info("クラスターの更新が停止されました")
}

// Node is 問い合わせノードの取得
func (s *RaftServer) Node() (*protobuf.Node, error) {
	nodes, err := s.Nodes()
	if err != nil {
		return nil, err
	}
	node, ok := nodes[s.nodeID]
	if !ok {
		return nil, errors.NotFound
	}
	node.State = s.StateStr()
	return node, nil
}

// Nodes is 全ノードの取得
func (s *RaftServer) Nodes() (map[string]*protobuf.Node, error) {
	cf := s.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		s.logger.Error(errors.RaftConfigration(err))
	}
	nodes := make(map[string]*protobuf.Node, 0)
	for _, server := range cf.Configuration().Servers {
		metadata, _ := s.getMetadata(string(server.ID))
		nodes[string(server.ID)] = &protobuf.Node{
			RaftAddress: string(server.Address),
			Metadata:    metadata,
		}
	}
	return nodes, nil
}

func (s *RaftServer) getMetadata(id string) (*protobuf.Metadata, error) {
	data := &protobuf.GetMetadataRequest{
		Id: id,
	}

	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(data, dataAny); err != nil {
		s.logger.Logrus().WithField("id", id).Error(err)
		return nil, err
	}

	event := &protobuf.Event{
		Type: protobuf.Event_GetNode,
		Data: dataAny,
	}

	msg, err := proto.Marshal(event)
	if err != nil {
		// s.logger.Error("failed to marshal the command into the bytes as message", zap.String("id", id), zap.Any("metadata", metadata), zap.Error(err))
		return nil, err
	}

	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		// s.logger.Error("failed to apply message bytes", zap.Duration("timeout", timeout), zap.Error(future.Error()))
		return nil, future.Error()
	} else {
		future.Response()
	}
	return nil, nil
}

// Join is ノードの追加
func (s *RaftServer) Join(id string, node *protobuf.Node) error {
	exist, err := s.Exist(id)
	if err != nil {
		return err
	}
	if !exist {
		if future := s.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(node.RaftAddress), 0, 0); future.Error() != nil {
			s.logger.Error(errors.AddVoter(err))
			return err
		}
		s.logger.Infof("ノートの参加が正常に処理されました[id:%s][address:%s]", id, node.RaftAddress)
	}
	if err := s.setMetadata(id, node.Metadata); err != nil {
		return err
	}

	if exist {
		return errors.NodeAlreadyExists
	}

	return nil
}

func (s *RaftServer) setMetadata(id string, metadata *protobuf.Metadata) error {
	data := &protobuf.SetMetadataRequest{
		Id:       id,
		Metadata: metadata,
	}

	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(data, dataAny); err != nil {
		s.logger.Error(errors.CommandUnmashal(err, id, metadata))
		return err
	}

	event := &protobuf.Event{
		Type: protobuf.Event_JoinNode,
		Data: dataAny,
	}

	msg, err := proto.Marshal(event)
	if err != nil {
		// s.logger.Error("failed to marshal the command into the bytes as message", zap.String("id", id), zap.Any("metadata", metadata), zap.Error(err))
		return err
	}

	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		// s.logger.Error("failed to apply message bytes", zap.Duration("timeout", timeout), zap.Error(future.Error()))
		return future.Error()
	}
	return nil
}

// Leave is ノードの離脱
func (s *RaftServer) Leave(id string) error {
	exist, err := s.Exist(id)
	if err != nil {
		return err
	}

	if exist {
		if future := s.raft.RemoveServer(raft.ServerID(id), 0, 0); future.Error() != nil {
			s.logger.Errorf("ノードの削除に失敗しました [id: %s] [err: %s]", id, future.Error())
			return future.Error()
		}
		s.logger.Info("ノードが正常に終了しました [id: %s]", id)
	}

	if err = s.deleteMetadata(id); err != nil {
		return err
	}

	if !exist {
		return errors.NodeDoesNotExist
	}

	return nil
}

func (s *RaftServer) deleteMetadata(id string) error {
	data := &protobuf.DeleteMetadataRequest{
		Id: id,
	}

	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(data, dataAny); err != nil {
		// s.logger.Error("failed to unmarshal request to the command data", zap.String("id", id), zap.Error(err))
		return err
	}

	event := &protobuf.Event{
		Type: protobuf.Event_LeaveNode,
		Data: dataAny,
	}

	msg, err := proto.Marshal(event)
	if err != nil {
		// s.logger.Error("failed to marshal the command into the bytes as the message", zap.String("id", id), zap.Error(err))
		return err
	}

	timeout := 60 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		// s.logger.Error("failed to apply message bytes", zap.Duration("timeout", timeout), zap.Error(future.Error()))
		return future.Error()
	}

	return nil
}
