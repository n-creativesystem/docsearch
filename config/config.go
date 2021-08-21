package config

import "path/filepath"

type DictionaryDB int

const (
	BadgerDB DictionaryDB = iota
	GRPC
)

type AppConfig struct {
	nodeId        string
	rootDirectory string
	dictionaryDB  DictionaryDB
}

func (conf *AppConfig) GetNodeID() string {
	return conf.nodeId
}

func (conf *AppConfig) GetRootDirectory() string {
	return conf.rootDirectory
}

func (conf *AppConfig) GetDocSearchIndexDirectory() string {
	return filepath.Join(conf.GetRootDirectory(), "docsearch_index")
}

func (conf *AppConfig) GetDictionaryDB() DictionaryDB {
	return conf.dictionaryDB
}

type Option func(config *AppConfig)

func WithNodeId(nodeId string) Option {
	return func(config *AppConfig) {
		config.nodeId = nodeId
	}
}

func WithDirectory(path string) Option {
	return func(config *AppConfig) {
		config.rootDirectory = path
	}
}

func WithDictionaryDB(name string) Option {
	return func(config *AppConfig) {
		switch name {
		default:
			config.dictionaryDB = BadgerDB
		}
	}
}

func New(opts ...Option) *AppConfig {
	conf := &AppConfig{}
	for _, opts := range opts {
		opts(conf)
	}
	return conf
}
