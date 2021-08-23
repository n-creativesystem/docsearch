package storage

import (
	"bytes"
	"io"
	"sync"

	"github.com/blugelabs/bluge/index"
	segment "github.com/blugelabs/bluge_segment_api"
)

type snapshotImpl struct {
	mu   sync.Mutex
	kind map[string]map[uint64][]byte
}

var _ index.Directory = (*snapshotImpl)(nil)

func newSnapshot() index.Directory {
	return &snapshotImpl{
		kind: map[string]map[uint64][]byte{},
	}
}

func (s *snapshotImpl) Persist(kind string, id uint64, w index.WriterTo, closeCh chan struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.kind[kind]; !ok {
		s.kind[kind] = map[uint64][]byte{}
	}
	buf := &bytes.Buffer{}
	if _, err := w.WriteTo(buf, closeCh); err != nil {
		return err
	}
	s.kind[kind][id] = buf.Bytes()
	return nil
}

func (s *snapshotImpl) Setup(readOnly bool) error {
	return nil
}

func (s *snapshotImpl) List(kind string) ([]uint64, error) {
	return nil, nil
}

func (s *snapshotImpl) Load(kind string, id uint64) (*segment.Data, io.Closer, error) {
	return nil, nil, nil
}

func (s *snapshotImpl) Remove(kind string, id uint64) error {
	return nil
}

func (s *snapshotImpl) Stats() (numItems uint64, numBytes uint64) {
	return 0, 0
}

func (s *snapshotImpl) Sync() error {
	return nil
}

func (s *snapshotImpl) Lock() error {
	return nil
}

func (s *snapshotImpl) Unlock() error {
	return nil
}
