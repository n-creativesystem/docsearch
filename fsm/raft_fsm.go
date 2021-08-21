package fsm

import (
	"encoding/json"
	"io"
	"sync"

	originalBadgerDB "github.com/dgraph-io/badger/v3"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/ikawaha/kagome-dict/dict"
	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/config"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/plugin/dictionary"
	"github.com/n-creativesystem/docsearch/plugin/dictionary/badger"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
	"github.com/n-creativesystem/docsearch/utils"
)

type RaftFSM struct {
	logger logger.DefaultLogger
	// index             storage.Index
	metadata          map[string]*protobuf.Metadata
	nodesMutex        sync.RWMutex
	documentFunctions *documentFunctions
	dictionaryStore   dictionary.Factory

	ApplyCh chan *protobuf.Event
}

var _ raft.FSM = (*RaftFSM)(nil)

func NewRaftFSM(conf *config.AppConfig, logger logger.WriteLogger) (*RaftFSM, error) {
	closeFlg := false
	index, err := storage.New(conf.GetDocSearchIndexDirectory(), logger)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeFlg {
			index.Close()
		}
	}()
	var dictionaryStore dictionary.Factory
	switch conf.GetDictionaryDB() {
	default:
		dictionaryStore = badger.New()
	}
	if err := dictionaryStore.Initialize(conf, logger); err != nil {
		closeFlg = true
		return nil, err
	}
	return &RaftFSM{
		logger:   logger,
		metadata: make(map[string]*protobuf.Metadata, 0),
		documentFunctions: &documentFunctions{
			index:  index,
			logger: logger,
		},
		ApplyCh:         make(chan *protobuf.Event, 1024),
		dictionaryStore: dictionaryStore,
	}, nil
}

func (f *RaftFSM) Start() error {
	registerModels, err := f.dictionaryStore.RegisterDictionary()
	if err != nil && err != originalBadgerDB.ErrKeyNotFound {
		return err
	}
	for _, registerModel := range registerModels {
		d, err := registerModel.NewUserDict()
		if err != nil {
			return err
		}
		a, err := analyzer.UserDic(d)
		if err != nil {
			return err
		}
		analyzer.Register(registerModel.Name, a)
	}
	return nil
}

// Apply is raftコマンドの実行
func (f *RaftFSM) Apply(log *raft.Log) interface{} {
	var event protobuf.Event
	err := proto.Unmarshal(log.Data, &event)
	if err != nil {
		f.logger.Error(err)
		return err
	}
	data, err := f.convertData(&event)
	if err != nil {
		return NewApplyErr(err)
	}
	switch event.Type {
	case protobuf.Event_JoinNode:
		req := data.(*protobuf.SetMetadataRequest)
		if err := f.setMetadata(req.Id, req.Metadata); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event
		return &ApplyResponse{}
	case protobuf.Event_LeaveNode:
		req := data.(*protobuf.DeleteMetadataRequest)
		if err := f.deleteMetadata(req.Id); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event
		return &ApplyResponse{}
	case protobuf.Event_GetNode:
		req := data.(*protobuf.GetMetadataRequest)
		metadata := f.getMetadata(req.Id)
		f.ApplyCh <- &event
		return &ApplyResponse{Data: metadata}
	case protobuf.Event_Upload:
		value := data.(*protobuf.Documents)
		if err := f.documentFunctions.update(value); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event

		return &ApplyResponse{}
	case protobuf.Event_Delete:
		value := data.(*protobuf.DeleteDocuments)
		if err := f.documentFunctions.bulkDelete(value); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event

		return &ApplyResponse{}
	case protobuf.Event_Search:
		req := data.(*protobuf.SearchRequest)
		var resp *storage.SearchResponse
		if resp, err = f.documentFunctions.search(req); err != nil {
			return NewApplyErr(err)
		}
		f.ApplyCh <- &event

		return &ApplyResponse{Data: resp}
	case protobuf.Event_Dictionary:
		value := data.(*protobuf.UserDictionaryRecords)
		dicRecords := make(dict.UserDictRecords, len(value.Records))
		for idx, record := range value.Records {
			dict := dict.UserDicRecord{
				Text:   record.Text,
				Tokens: make([]string, len(record.Tokens)),
				Yomi:   make([]string, len(record.Yomi)),
				Pos:    record.Pos,
			}
			copy(dict.Tokens, record.Tokens)
			copy(dict.Yomi, record.Yomi)
			dicRecords[idx] = dict
		}
		buf, err := json.Marshal(dicRecords)
		if err != nil {
			return NewApplyErr(err)
		}
		a, err := analyzer.UserDic(dicRecords)
		if err != nil {
			return NewApplyErr(err)
		}
		if err := f.dictionaryStore.Save(value.Id, buf); err != nil {
			return NewApplyErr(err)
		}
		analyzer.Register(value.Id, a)
		f.ApplyCh <- &event

		return &ApplyResponse{}
	case protobuf.Event_RemoveDictionary:
		value := data.(*protobuf.DeleteDictionaryRequest)
		if err := f.dictionaryStore.Delete(value.Id); err != nil {
			return NewApplyErr(err)
		}
		analyzer.Remove(value.Id)
		f.ApplyCh <- &event

		return &ApplyResponse{}
	}
	return nil
}

// GetDictionary is ユーザー辞書の取得
func (f *RaftFSM) getDictionary(name string) (*dict.UserDict, error) {
	buf, err := f.dictionaryStore.Get(name)
	if err != nil {
		return nil, err
	}
	var dicts dict.UserDictRecords
	if err := json.Unmarshal(buf, &dicts); err != nil {
		return nil, err
	}
	return dicts.NewUserDict()
}

// convertData is raft コマンドの生成
func (f *RaftFSM) convertData(event *protobuf.Event) (interface{}, error) {
	data, err := utils.MarshalAny(event.Data)
	if err != nil {
		f.logger.Errorf("failed to marshal event data to set request: %s", event.Type.String())
		return nil, err
	}
	if data == nil {
		err = errors.Nil
		f.logger.Errorf("request is nil: %s", event.Type.String())
		return nil, err
	}
	return data, nil
}

// setMetadata is メタデータのセット
func (f *RaftFSM) setMetadata(id string, metadata *protobuf.Metadata) error {
	f.nodesMutex.Lock()
	defer f.nodesMutex.Unlock()

	f.metadata[id] = metadata

	return nil
}

// deleteMetadata is メタデータの削除
func (f *RaftFSM) deleteMetadata(id string) error {
	f.nodesMutex.Lock()
	defer f.nodesMutex.Unlock()
	delete(f.metadata, id)
	return nil
}

// Snapshot is 未実装
func (f *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &FSMSnapshot{
		index:  f.documentFunctions.index,
		logger: f.logger,
	}, nil
}

// Restore is 未実装
func (f *RaftFSM) Restore(rc io.ReadCloser) error {
	return nil
}

// GetMetadata メタデータの取得
func (f *RaftFSM) getMetadata(id string) *protobuf.Metadata {
	if metadata, exists := f.metadata[id]; exists {
		return metadata
	} else {
		f.logger.Debugf("metadata not found: %s", id)
		return nil
	}
}

// Close is 終了処理
// ユーザー辞書DB、ドキュメントDBのClose
func (f *RaftFSM) Close() error {
	f.dictionaryStore.Close()
	return f.documentFunctions.Close()
}
