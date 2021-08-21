package badger

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/n-creativesystem/docsearch/config"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/plugin/dictionary"
)

type badgerFactory struct {
	store *Factory
	once  sync.Once
}

// シングルトン
var bf badgerFactory

type Factory struct {
	logger     logger.DefaultLogger
	store      *badger.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func New() *Factory {
	bf.once.Do(func() {
		bf.store = &Factory{}
	})
	return bf.store
}

func (f *Factory) Initialize(conf *config.AppConfig, logger logger.DefaultLogger) error {
	f.logger = logger
	path := filepath.Join(conf.GetRootDirectory(), "dictionary_badger_db")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	badgerOpts := badger.DefaultOptions(path)
	badgerOpts.ValueDir = path
	badgerOpts.SyncWrites = false
	badgerOpts.Logger = nil
	store, err := badger.Open(badgerOpts)
	f.store = store
	f.ctx, f.cancelFunc = context.WithCancel(context.Background())
	go f.runGC()
	return err
}

func (f *Factory) RegisterDictionary() (registerModel []dictionary.RegisterModel, err error) {
	err = f.store.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			var err error
			val := []byte{}
			name := []byte{}
			name = it.Item().KeyCopy(name)
			item := it.Item()
			if val, err = item.ValueCopy(val); err != nil {
				return err
			}
			registerModel = append(registerModel, dictionary.RegisterModel{
				Name:       string(name),
				Dictionary: val,
			})
		}
		return nil
	})
	if err != nil {
		registerModel = []dictionary.RegisterModel{}
	}
	return
}

func (f *Factory) Close() error {
	f.cancelFunc()
	return f.store.Close()
}

func (f *Factory) Save(name string, dictionaries []byte) error {
	return f.store.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(name), dictionaries); err != nil {
			return err
		}
		return nil
	})
}

func (f *Factory) Get(name string) (dictionaries []byte, err error) {
	err = f.store.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(name))
		if err != nil {
			return err
		}
		if dictionaries, err = item.ValueCopy(dictionaries); err != nil {
			return err
		}
		return nil
	})
	return
}

func (f *Factory) Delete(name string) error {
	return f.store.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(name))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return nil
	})
}

func (f *Factory) runGC() {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			err := f.store.RunValueLogGC(0.5)
			if err != nil {
				if err == badger.ErrNoRewrite {
					f.logger.Debugf("BadgerDB GCは発生しませんでした: %s", err)
				} else {
					f.logger.Errorf("BadgerDB GCでエラーが発生しました: %v", err)
				}
			}
		case <-f.ctx.Done():
			return
		}
	}
}
