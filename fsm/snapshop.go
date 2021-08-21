package fsm

import (
	"github.com/hashicorp/raft"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/storage"
)

type FSMSnapshot struct {
	index  storage.Index
	logger logger.DefaultLogger
}

var _ raft.FSMSnapshot = (*FSMSnapshot)(nil)

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	// start := time.Now()
	f.logger.Info("スナップショット作成を開始します")
	// defer func() {
	// 	if err := sink.Close(); err != nil {
	// 		f.logger.Errorf("sinkのクローズに失敗しました: %s", err)
	// 	}
	// }()
	// cancel := make(chan struct{})
	// return f.index.Snapshot("", cancel)
	return nil
}

func (f *FSMSnapshot) Release() {
	f.logger.Info("Release")
}
