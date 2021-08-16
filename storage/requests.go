package storage

import "github.com/blugelabs/bluge"

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
	v := GetAnalyzer(name)
	if v != nil {
		q.MatchQuery.SetAnalyzer(v)
	}
	return q
}
