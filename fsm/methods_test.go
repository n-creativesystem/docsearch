package fsm

import (
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/blugelabs/bluge"
	segment "github.com/blugelabs/bluge_segment_api"
	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/model"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/n-creativesystem/docsearch/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockIndex struct {
	t *testing.T
}

var _ storage.Index = (*mockIndex)(nil)

func (i *mockIndex) Close() error {
	return nil
}

func (i *mockIndex) Search(request *bluge.TopNSearch) (*storage.SearchResponse, error) {
	query := storage.NewMatchQuery("contents").SetField("contents").SetAnalyzer(analyzer.GetAnalyzer("kagome"))
	filter := []bluge.Query{}
	q := bluge.NewBooleanQuery().AddMust(query).AddMust(filter...)
	expected := bluge.NewTopNSearch(10, q).WithStandardAggregations().SetFrom(10).ExplainScores()
	if reflect.DeepEqual(request, expected) {
		return nil, errors.New("not deep equal")
	}
	return &storage.SearchResponse{
		Metadata:  storage.Metadata{},
		Documents: []*storage.DocumentMatch{},
	}, nil
}

func (i *mockIndex) Update(documents []*bluge.Document) error {
	doc := documents[0]
	expected := bluge.NewDocument("test")
	expected.AddField(bluge.NewKeywordField("contents", "contents").WithAnalyzer(analyzer.GetAnalyzer("kagome")).StoreValue().HighlightMatches())
	expected.AddField(bluge.NewKeywordField("title", "test").WithAnalyzer(analyzer.GetAnalyzer("kagome")))
	var err error
	doc.EachField(func(df segment.Field) {
		expected.EachField(func(ef segment.Field) {
			if df.Name() == ef.Name() {
				if !reflect.DeepEqual(df, ef) {
					err = errors.New("not deep equal: %s, %s", df.Name(), ef.Name())
				}
			}
		})
	})
	return err
}

func (i *mockIndex) Delete(id string) error {
	return nil
}

func (i *mockIndex) BulkDelete(ids []string) error {
	return nil
}

func (i *mockIndex) Snapshot(writer io.Writer, id string, cancel chan struct{}) error {
	return nil
}

func (i *mockIndex) Restore(rc io.ReadCloser) error {
	return nil
}

func TestMethods(t *testing.T) {
	log := logrus.New()
	funcs := documentFunctions{
		index: &mockIndex{
			t: t,
		},
		logger: log,
	}
	fields := map[string]model.FieldMapping{
		"title": {
			Value:            "title",
			AnalyserName:     "kagome",
			Type:             model.Keyword,
			StoreValue:       false,
			HighlightMatches: false,
		},
		"contents": {
			Value:            "contents",
			AnalyserName:     "kagome",
			Type:             model.Text,
			StoreValue:       true,
			HighlightMatches: true,
		},
	}
	buf, _ := json.Marshal(&fields)
	updateValue := &protobuf.Documents{
		Requests: []*protobuf.Document{
			{
				Id:     "test",
				Fields: buf,
			},
		},
	}
	if err := funcs.update(updateValue); !assert.NoError(t, err) {
		return
	}
	searchReq := &protobuf.SearchRequest{
		Query: &protobuf.Query{
			Value:        "contents",
			Fields:       "contents",
			AnalyzerName: "kagome",
		},
	}
	if _, err := funcs.search(searchReq); !assert.NoError(t, err) {
		return
	}
}
