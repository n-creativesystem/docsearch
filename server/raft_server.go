package server

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	raftbadgerdb "github.com/BBVA/raft-badger"
	"github.com/blugelabs/bluge"
	"github.com/dgraph-io/badger/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/fsm"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
	"github.com/n-creativesystem/docsearch/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type RaftServer struct {
	nodeID        string
	raftAddress   string
	dataDirectory string
	bootstrap     bool
	raft          *raft.Raft
	fsm           *fsm.FSM
	transport     *raft.NetworkTransport
	logger        *logrus.Logger

	watchClusterStopCh chan struct{}
	watchClusterDoneCh chan struct{}

	applyCh chan *protobuf.Event
}

func NewRaftServer(id, raftAddress, dataDirectory string, bootstrap bool, log *logrus.Logger) (*RaftServer, error) {
	indexPath := filepath.Join(dataDirectory, "index")
	fsm, err := fsm.NewRaftFSM(indexPath, log)
	if err != nil {
		log.Errorf("create fsm instance error: %s", err.Error())
		return nil, err
	}
	return &RaftServer{
		nodeID:             id,
		raftAddress:        raftAddress,
		dataDirectory:      dataDirectory,
		bootstrap:          bootstrap,
		logger:             log,
		fsm:                fsm,
		watchClusterStopCh: make(chan struct{}),
		watchClusterDoneCh: make(chan struct{}),

		applyCh: make(chan *protobuf.Event, 1024),
	}, nil
}

func (s *RaftServer) Start() error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(s.nodeID)
	config.SnapshotThreshold = 1024
	addr, err := net.ResolveTCPAddr("tcp", s.raftAddress)
	if err != nil {
		s.logger.Errorf("resolver tcp address error: %s", err.Error())
		return err
	}
	s.transport, err = raft.NewTCPTransport(s.raftAddress, addr, 3, 10*time.Second, io.Discard)
	if err != nil {
		s.logger.Errorf("create transport instance error: %s", err.Error())
		return err
	}
	snapshotStore, err := raft.NewFileSnapshotStore(s.dataDirectory, 2, io.Discard)
	if err != nil {
		s.logger.Errorf("create snapshot store instance error: %s", err.Error())
		return err
	}
	logStorePath := filepath.Join(s.dataDirectory, "raft", "log")
	err = os.MkdirAll(logStorePath, 0755)
	if err != nil {
		s.logger.Errorf("mkdir error %s, %s", logStorePath, err.Error())
		return err
	}
	logStoreBadgerOpts := badger.DefaultOptions(logStorePath)
	logStoreBadgerOpts.ValueDir = logStorePath
	logStoreBadgerOpts.SyncWrites = false
	logStoreBadgerOpts.Logger = nil
	logStoreOpts := raftbadgerdb.Options{
		Path:          logStorePath,
		BadgerOptions: &logStoreBadgerOpts,
	}
	raftLogStore, err := raftbadgerdb.New(logStoreOpts)
	if err != nil {
		s.logger.Errorf("log store badgerDB error: %s", err.Error())
		return err
	}
	stableStorePath := filepath.Join(s.dataDirectory, "raft", "stable")
	err = os.MkdirAll(stableStorePath, 0755)
	if err != nil {
		s.logger.Errorf("mkdir error %s, %s", stableStorePath, err.Error())
		return err
	}
	stableStoreBadgerOpts := badger.DefaultOptions(stableStorePath)
	stableStoreBadgerOpts.ValueDir = stableStorePath
	stableStoreBadgerOpts.SyncWrites = false
	stableStoreBadgerOpts.Logger = nil
	stableStoreOpts := raftbadgerdb.Options{
		Path:          stableStorePath,
		BadgerOptions: &stableStoreBadgerOpts,
	}
	raftStableStore, err := raftbadgerdb.New(stableStoreOpts)
	if err != nil {
		s.logger.Errorf("stable store dadgerDB error: %s", err.Error())
		return err
	}
	s.raft, err = raft.NewRaft(config, s.fsm, raftLogStore, raftStableStore, snapshotStore, s.transport)
	if err != nil {
		s.logger.Errorf("create raft instace  error: %s", err.Error())
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
	s.logger.Infof("Raft server started: %s", s.raftAddress)
	return nil
}

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
				s.logger.Debugf("detected a leader address: %s", leaderAddr)
				return leaderAddr, nil
			}
		case <-timer.C:
			err := errors.TimeOut
			s.logger.Errorf("Leader address error: %s", err.Error())
			return "", err
		}
	}
}

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

func (s *RaftServer) Stop() error {
	if err := s.fsm.Close(); err != nil {
		s.logger.Error("failed to close FSM: %s", err.Error())
	}
	s.logger.Info("Raft FSM Closesd")
	if future := s.raft.Shutdown(); future.Error() != nil {
		err := future.Error()
		s.logger.Errorf("failed to shutdown Raft: %s", err.Error())
	}
	s.logger.Info("Raft has shutdown: %s", s.raftAddress)
	return nil
}

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

func (s *RaftServer) Leave(id string) error {
	exist, err := s.Exist(id)
	if err != nil {
		return err
	}

	if exist {
		if future := s.raft.RemoveServer(raft.ServerID(id), 0, 0); future.Error() != nil {
			// s.logger.Error("failed to remove server", zap.String("id", id), zap.Error(future.Error()))
			return future.Error()
		}
		// s.logger.Info("node has successfully left", zap.String("id", id))
	}

	if err = s.deleteMetadata(id); err != nil {
		return err
	}

	if !exist {
		return errors.NodeDoesNotExist
	}

	return nil
}

func (s *RaftServer) Upload(req *protobuf.Documents) error {
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_Upload,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	timeout := 69 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return future.Error()
	}
	return nil
}

func (s *RaftServer) Search(req *protobuf.SearchRequest) (*protobuf.SearchResponse, error) {
	query := storage.NewMatchQuery(req.Query)
	query.SetAnalyzer(req.AnalyzerName)
	for _, field := range req.Fields {
		query.SetField(field)
	}
	// query := bluge.NewMatchQuery(q).SetAnalyzer(ja.Analyzer()).SetField("body")
	sreq := bluge.NewTopNSearch(10, query).WithStandardAggregations()
	ids, err := s.fsm.Search(sreq)
	if err != nil {
		return nil, err
	}

	var books = make([]storage.Book, len(ids))
	i := 0
	for _, id := range ids {
		b, err := storage.Exists(id)
		if err == nil {
			books[i] = *b
			i++
		}
	}
	buf, _ := json.Marshal(&books)
	res := &protobuf.SearchResponse{
		Result: buf,
	}
	return res, nil
}

func (s *RaftServer) BulkDelete(req *protobuf.DeleteDocuments) error {
	dataAny := &any.Any{}
	if err := utils.UnmarshalAny(req, dataAny); err != nil {
		return err
	}
	event := &protobuf.Event{
		Type: protobuf.Event_Delete,
		Data: dataAny,
	}
	msg, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	timeout := 69 * time.Second
	if future := s.raft.Apply(msg, timeout); future.Error() != nil {
		return future.Error()
	}
	return nil
}

func (s *RaftServer) getMetadata(id string) (*protobuf.Metadata, error) {
	metadata := s.fsm.GetMetadata(id)
	if metadata == nil {
		return nil, errors.NotFound
	}
	return metadata, nil
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
		Type: protobuf.Event_Join,
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
		Type: protobuf.Event_Leave,
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

func (s *RaftServer) State() raft.RaftState {
	return s.raft.State()
}

func (s *RaftServer) StateStr() string {
	return s.State().String()
}

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

func (s *RaftServer) WaitForDetectLeader(timeout time.Duration) error {
	if _, err := s.LeaderAddress(timeout); err != nil {
		s.logger.Error("failed to wait for detect leader: %s", err.Error())
		return err
	}

	return nil
}

func (s *RaftServer) startWatchCluster(checkInterval time.Duration) {
	s.logger.Info("start to update cluster info")

	defer func() {
		close(s.watchClusterDoneCh)
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	timeout := 60 * time.Second
	if err := s.WaitForDetectLeader(timeout); err != nil {
		if err == errors.TimeOut {
			s.logger.Error("leader detection timed out", timeout, err)
		} else {
			s.logger.Error("failed to detect leader", err)
		}
	}

	for {
		select {
		case <-s.watchClusterStopCh:
			s.logger.Info("received a request to stop updating a cluster")
			return
		case <-s.raft.LeaderCh():
			s.logger.Info("became a leader", string(s.raft.Leader()))
		case event := <-s.fsm.ApplyCh:
			s.applyCh <- event
		case <-ticker.C:
			_ = s.raft.Stats()
		}
	}
}

func (s *RaftServer) stopWatchCluster() {
	if s.watchClusterStopCh != nil {
		s.logger.Info("send a request to stop updating a cluster")
		close(s.watchClusterStopCh)
	}

	s.logger.Info("wait for the cluster update to stop")
	<-s.watchClusterDoneCh
	s.logger.Info("the cluster update has been stopped")
}
