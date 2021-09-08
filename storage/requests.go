package storage

import (
	"github.com/blugelabs/bluge"
	"github.com/n-creativesystem/docsearch/analyzer"
)

type MatchQuery struct {
	*bluge.MatchQuery
}

func NewMatchQuery(q string) *MatchQuery {
	return &MatchQuery{
		MatchQuery: bluge.NewMatchQuery(q),
	}
}

func (q *MatchQuery) Build() bluge.Query {
	return q.MatchQuery
}

func (q *MatchQuery) SetAnalyzer(name string) *MatchQuery {
	v := analyzer.GetAnalyzer(name)
	if v != nil {
		q.MatchQuery.SetAnalyzer(v)
	}
	return q
}

type MatchAllQuery struct {
	*bluge.MatchAllQuery
}

func NewAll() *MatchAllQuery {
	return &MatchAllQuery{
		MatchAllQuery: bluge.NewMatchAllQuery(),
	}
}

type Term struct {
	*bluge.TermQuery
}

func NewTerm(field, value string) *Term {
	return &Term{
		TermQuery: bluge.NewTermQuery(value).SetField(field),
	}
}

type WildCard struct {
	*bluge.WildcardQuery
}

func NewWildCard(field, value string) *WildCard {
	return &WildCard{
		WildcardQuery: bluge.NewWildcardQuery(value).SetField(field),
	}
}
