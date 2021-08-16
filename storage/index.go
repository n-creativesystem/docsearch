package storage

import (
	"context"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/sirupsen/logrus"
)

type Index struct {
	writer *bluge.Writer
	logger *logrus.Logger
}

func New(directory string, logger *logrus.Logger) (*Index, error) {
	config := bluge.DefaultConfig(directory)
	// conf := index.DefaultConfig(directory)

	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return nil, err
	}
	return &Index{
		writer: writer,
		logger: logger,
	}, nil
}

func (i *Index) Close() error {
	return i.writer.Close()
}

func (i *Index) reader(f func(reader *bluge.Reader) error) error {
	reader, err := i.writer.Reader()
	if err != nil {
		i.logger.Errorf("reader error: %s", err.Error())
		return err
	}
	defer reader.Close()
	return f(reader)
}

func (i *Index) Search(request bluge.SearchRequest) ([]string, error) {
	ids := []string{}
	err := i.reader(func(reader *bluge.Reader) error {
		cnt, _ := reader.Count()
		i.logger.Infof("count: %d", cnt)
		fields, _ := reader.Fields()
		i.logger.Infof("fields: %v", fields)
		documentMatchIterator, err := reader.Search(context.Background(), request)
		if err != nil {
			return err
		}
		match, err := documentMatchIterator.Next()
		for err == nil && match != nil {
			err = match.VisitStoredFields(func(field string, value []byte) bool {
				if field == "_id" {
					ids = append(ids, string(value))
					i.logger.Infof("match: %s", string(value))
				}
				return true
			})
			if err != nil {
				i.logger.Errorf("document request error: %s", err.Error())
				return err
			}
			match, err = documentMatchIterator.Next()
		}
		return nil
	})
	return ids, err
}

func (i *Index) Update(documents []*bluge.Document) error {
	batch := index.NewBatch()
	for _, doc := range documents {
		doc := doc
		batch.Update(doc.ID(), doc)
	}
	return i.writer.Batch(batch)
}

func (i *Index) Delete(id string) error {
	return i.BulkDelete([]string{id})
}

func (i *Index) BulkDelete(ids []string) error {
	batch := index.NewBatch()
	for _, id := range ids {
		id := id
		batch.Delete(bluge.Identifier(id))
	}
	return i.writer.Batch(batch)
}
