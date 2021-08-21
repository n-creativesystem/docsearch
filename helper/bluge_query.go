package helper

import (
	"github.com/blugelabs/bluge"
	"github.com/n-creativesystem/docsearch/analyzer"
)

type queryConfig struct {
	analyzerName string
}

type Option func(conf *queryConfig)

func WithAnalyzer(name string) Option {
	return func(conf *queryConfig) {
		conf.analyzerName = name
	}
}

func getConfig(opts ...Option) *queryConfig {
	conf := &queryConfig{}
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

// MatchPhrase is フレーズ一致検索(テナント, インデックス)
func MatchPhrase(field, value string, opts ...Option) bluge.Query {
	conf := getConfig(opts...)
	query := bluge.NewMatchPhraseQuery(value).SetField(field)
	if conf.analyzerName != "" {
		if v := analyzer.GetAnalyzer(conf.analyzerName); v != nil {
			query.SetAnalyzer(v)
		}
	}
	return query
}

// TermFunc is 用語検索
func TermFunc(field, value string, opts ...Option) bluge.Query {
	_ = getConfig(opts...)
	query := bluge.NewTermQuery(value).SetField(field)
	return query
}
