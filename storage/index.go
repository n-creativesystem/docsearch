package storage

import (
	"context"
	"io"
	"log"
	"math"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/bluge/search"
	"github.com/n-creativesystem/docsearch/logger"
)

type Index interface {
	Close() error
	Search(request *bluge.TopNSearch) (*SearchResponse, error)
	Update(documents []*bluge.Document) error
	Delete(id string) error
	BulkDelete(ids []string) error
	Snapshot(writer io.Writer, id string) error
	Restore(rc io.ReadCloser) error
}

type store struct {
	directory string
	config    bluge.Config
	writer    *bluge.Writer
	logger    logger.DefaultLogger
}

func New(directory string, logger logger.WriteLogger) (Index, error) {
	config := bluge.DefaultConfig(directory)
	config.Logger = log.New(logger, "bluge", log.LstdFlags)
	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return nil, err
	}
	return &store{
		directory: directory,
		config:    config,
		writer:    writer,
		logger:    logger,
	}, nil
}

func (s *store) Close() error {
	return s.writer.Close()
}

func (s *store) reader(f func(reader *bluge.Reader) error) error {
	reader, err := bluge.OpenReader(s.config)
	if err != nil {
		s.logger.Errorf("reader error: %s", err.Error())
		return err
	}
	defer reader.Close()
	return f(reader)
}

type SearchResponse struct {
	Metadata  Metadata         `json:"metadata"`
	Documents []*DocumentMatch `json:"documents"`
}

type Metadata struct {
	NextPage     int64   `json:"next_page,omitempty"`
	PreviousPage int64   `json:"previous_page,omitempty"`
	TopScore     float64 `json:"top_score"`
	Total        uint64  `json:"total"`
}

func (m *Metadata) CalcPage(searchSize int, nowPage int64) {
	numPages := int64(math.Ceil(float64(m.Total) / float64(searchSize)))
	if numPages > nowPage {
		m.NextPage = nowPage + 1
	}
	if nowPage != 1 {
		m.PreviousPage = nowPage - 1
	}
}

type DocumentMatch struct {
	Document interface{}         `json:"document"`
	Score    float64             `json:"score"`
	Expl     *search.Explanation `json:"explanation"`
	ID       string              `json:"id"`
}

func (s *store) Search(request *bluge.TopNSearch) (*SearchResponse, error) {
	resp := &SearchResponse{
		Metadata:  Metadata{},
		Documents: []*DocumentMatch{},
	}
	err := s.reader(func(reader *bluge.Reader) error {
		var documentMatchs []*DocumentMatch
		documentMatchIterator, err := reader.Search(context.Background(), request)
		if err != nil {
			return err
		}
		next, err := documentMatchIterator.Next()
		for err == nil && next != nil {
			var doc DocumentMatch
			mp := map[string]interface{}{}
			err = next.VisitStoredFields(func(field string, value []byte) bool {
				if field == "_id" {
					doc.ID = string(value)
				} else {
					mp[field] = string(value)
				}
				return true
			})
			if err != nil {
				s.logger.Errorf("document request error: %s", err.Error())
				return err
			}
			doc.Document = mp
			doc.Score = next.Score
			doc.Expl = next.Explanation
			documentMatchs = append(documentMatchs, &doc)
			next, err = documentMatchIterator.Next()
			documentMatchIterator.Aggregations()
		}
		aggs := documentMatchIterator.Aggregations()
		count := aggs.Count()
		topScore := aggs.Metric("max_score")
		resp.Metadata.TopScore = topScore
		resp.Metadata.Total = count
		resp.Documents = make([]*DocumentMatch, len(documentMatchs))
		copy(resp.Documents, documentMatchs)
		return nil
	})
	return resp, err
}

func (s *store) Update(documents []*bluge.Document) error {
	batch := index.NewBatch()
	for _, doc := range documents {
		doc := doc
		batch.Update(doc.ID(), doc)
	}
	return s.writer.Batch(batch)
}

func (s *store) Delete(id string) error {
	return s.BulkDelete([]string{id})
}

func (s *store) BulkDelete(ids []string) error {
	batch := index.NewBatch()
	for _, id := range ids {
		id := id
		batch.Delete(bluge.Identifier(id))
	}
	return s.writer.Batch(batch)
}

func (s *store) Snapshot(writer io.Writer, id string) error {
	// snapShotPath := filepath.Join(s.directory, "snapshot", id)
	// return s.reader(func(reader *bluge.Reader) error {
	// 	return reader.Backup(snapShotPath, nil)
	// })
	// inMemory := index.NewInMemoryDirectory()
	// reader, _ := index.OpenReader(index.DefaultConfigWithDirectory(func() index.Directory {
	// 	return inMemory
	// }))
	// if err := reader.Backup(inMemory, nil); err != nil {
	// 	return err
	// }
	// if ids, err := inMemory.List(index.ItemKindSegment); err != nil {
	// 	return err
	// } else {
	// 	for _, id := range ids {
	// 		data, _, _ := inMemory.Load(index.ItemKindSegment, id)
	// 		if data != nil {
	// 			if _, err := data.WriteTo(writer); err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// }
	return nil
}

func (s *store) Restore(rc io.ReadCloser) error {
	// buf, _ := io.ReadAll(rc)
	// s.logger.Info(string(buf))
	return nil
}
