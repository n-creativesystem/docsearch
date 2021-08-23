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
	f.logger.Info("スナップショット作成を開始します")
	// f.logger.Infof("id: %s", sink.ID())
	// if err := f.index.Snapshot(sink, sink.ID()); err != nil {
	// 	return err
	// }
	return nil
}

func (f *FSMSnapshot) Release() {
	f.logger.Info("Release")
}
