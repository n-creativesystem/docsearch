package badgerdb

import (
	"os"
	"path/filepath"

	raftbadgerdb "github.com/BBVA/raft-badger"
	"github.com/dgraph-io/badger/v3"
)

func New(rootDirectory string) (*raftbadgerdb.BadgerStore, error) {
	path := filepath.Join(rootDirectory, "raft_log_stable")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}
	badgerOpts := badger.DefaultOptions(path)
	badgerOpts.ValueDir = path
	badgerOpts.SyncWrites = false
	badgerOpts.Logger = nil
	storeOpts := raftbadgerdb.Options{
		Path:          path,
		BadgerOptions: &badgerOpts,
		ValueLogGC:    true,
	}
	store, err := raftbadgerdb.New(storeOpts)
	if err != nil {
		return nil, err
	}
	return store, nil
}
