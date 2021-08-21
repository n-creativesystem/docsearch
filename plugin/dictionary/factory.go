package dictionary

import (
	"encoding/json"

	"github.com/ikawaha/kagome-dict/dict"
	"github.com/n-creativesystem/docsearch/config"
	"github.com/n-creativesystem/docsearch/logger"
)

type RegisterModel struct {
	Name       string
	Dictionary []byte
}

func (r *RegisterModel) NewUserDict() (dict.UserDictRecords, error) {
	var records dict.UserDictRecords
	err := json.Unmarshal(r.Dictionary, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

type Factory interface {
	Initialize(appConfig *config.AppConfig, logger logger.DefaultLogger) error
	RegisterDictionary() (registerModel []RegisterModel, err error)
	Close() error
	Save(name string, dictionaries []byte) error
	Get(name string) (dictionaries []byte, err error)
	Delete(name string) error
}
