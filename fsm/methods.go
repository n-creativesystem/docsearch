package fsm

import (
	"encoding/json"
	"path/filepath"

	"github.com/blugelabs/bluge"
	"github.com/n-creativesystem/docsearch/helper"
	"github.com/n-creativesystem/docsearch/logger"
	"github.com/n-creativesystem/docsearch/model"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
)

type documentFunctions struct {
	// index  storage.Index
	indexDirectory string
	logger         logger.WriteLogger
}

// search is ドキュメント検索
func (f *documentFunctions) search(req *protobuf.SearchRequest) (*storage.SearchResponse, error) {
	// 全文検索する内容
	queries := make([]bluge.Query, 0, len(req.Query))
	var err error
	for field, value := range req.Query {
		if q, err := helper.GetQuery(field, value); err != nil {
			return nil, err
		} else {
			queries = append(queries, q)
		}
	}
	var page int = 0
	var size int = 0
	if req.Metadata != nil {
		page = int(req.Metadata.From)
		size = int(req.Metadata.Size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size
	query := bluge.NewBooleanQuery().AddMust(queries...)
	searchRequest := bluge.NewTopNSearch(size, query).WithStandardAggregations().SetFrom(offset).ExplainScores()
	var resp *storage.SearchResponse
	err = f.index(req.Tenant, func(index storage.Index) error {
		var err error
		resp, err = index.Search(searchRequest)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.Metadata.CalcPage(size, page)
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
	return f.index(value.Tenant, func(index storage.Index) error {
		return index.Update(docs)
	})
}

func (f *documentFunctions) bulkDelete(req *protobuf.DeleteDocuments) error {
	var ids = make([]string, len(req.Requests))
	for i, r := range req.Requests {
		ids[i] = r.Id
	}
	return f.index(req.Tenant, func(index storage.Index) error {
		return index.BulkDelete(ids)
	})
}

// func (f *documentFunctions) Close() error {
// 	return f.index.Close()
// }

func (f *documentFunctions) index(tenant string, fn func(index storage.Index) error) error {
	index, err := storage.New(filepath.Join(f.indexDirectory, tenant), f.logger)
	if err != nil {
		return err
	}
	defer index.Close()
	return fn(index)
}
