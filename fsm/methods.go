package fsm

import (
	"encoding/json"
	"fmt"

	"github.com/blugelabs/bluge"
	querystr "github.com/blugelabs/query_string"
	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/helper"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/model"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
)

type documentFunctions struct {
	index  storage.Index
	logger logger.DefaultLogger
}

// search is ドキュメント検索
func (f *documentFunctions) search(req *protobuf.SearchRequest) (*storage.SearchResponse, error) {
	// 全文検索する内容
	var match bluge.Query
	var err error
	if req.BleveFormat {
		match, err = querystr.ParseQueryString(req.Query.Value, querystr.DefaultOptions())
		if err != nil {
			return nil, fmt.Errorf("errror parsing query string '%s': %v", req.Query.Value, err)
		}
	} else {
		query := storage.NewMatchQuery(req.Query.Value).SetField(req.Query.Fields)
		analyzer := analyzer.GetAnalyzer(req.Query.AnalyzerName)
		if analyzer != nil {
			query = query.SetAnalyzer(analyzer)
		}
		match = query
	}
	filter := []bluge.Query{
		// tenant
		// index
		// type
	}
	for _, term := range req.TermQuery {
		termFilter := helper.TermFunc(term.Field, term.Value)
		filter = append(filter, termFilter)
	}
	if req.Page < 1 {
		req.Page = 1
	}
	resultPerPage := 10
	offset := (req.Page - 1) * int64(resultPerPage)
	q := bluge.NewBooleanQuery().AddMust(match).AddMust(filter...)
	searchRequest := bluge.NewTopNSearch(resultPerPage, q).WithStandardAggregations().SetFrom(int(offset)).ExplainScores()
	resp, err := f.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	resp.Metadata.CalcPage(resultPerPage, req.Page)
	return resp, nil
}

// update is ドキュメントの追加・更新
func (f *documentFunctions) update(value *protobuf.Documents) error {
	docs := make([]*bluge.Document, 0, 100)
	req := value.Requests
	for _, r := range req {
		var fields map[string]model.FieldMapping
		if err := json.Unmarshal(r.Fields, &fields); err != nil {
			f.logger.Error(err)
			continue
		}
		doc := bluge.NewDocument(r.Id)
		for fieldName, v := range fields {
			field, err := v.BlugeField(fieldName)
			if err != nil {
				return err
			}
			doc.AddField(field)
		}
		docs = append(docs, doc)
	}
	return f.index.Update(docs)
}

func (f *documentFunctions) bulkDelete(req *protobuf.DeleteDocuments) error {
	var ids = make([]string, len(req.Requests))
	for i, r := range req.Requests {
		ids[i] = r.Id
	}
	return f.index.BulkDelete(ids)
}

func (f *documentFunctions) Close() error {
	return f.index.Close()
}
