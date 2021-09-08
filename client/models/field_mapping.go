package models

import (
	"encoding/json"

	"github.com/n-creativesystem/docsearch/protobuf"
)

type FieldMapping struct {
	Value            interface{} `json:"value"`
	AnalyserName     string      `json:"analyzer_name"`
	Type             string      `json:"type"`
	StoreValue       bool        `json:"store_value"`
	HighlightMatches bool        `json:"highlight_matches"`
}

type MappingType string

const (
	text         = "text"
	keyword      = "keyword"
	numeric      = "numeric"
	numericRange = "numeric_range"
	geopoint     = "geopoint"
	dateTime     = "datetime"

	Text         = text
	Keyword      = keyword
	Numeric      = numeric
	NumericRange = numericRange
	Geopoint     = geopoint
	DateTime     = dateTime
)

type Document struct {
	DocumentId string
	Fields     map[string]FieldMapping
}

type Documents []Document

func (f Documents) Convert(tenant string) (*protobuf.Documents, error) {
	docs := &protobuf.Documents{
		Requests: make([]*protobuf.Document, 0, len(f)),
	}
	docs.Tenant = tenant
	for _, document := range f {
		buf, err := json.Marshal(document.Fields)
		if err != nil {
			return nil, err
		}
		doc := &protobuf.Document{
			Tenant: tenant,
			Id:     document.DocumentId,
			Fields: buf,
		}
		docs.Requests = append(docs.Requests, doc)
	}
	return docs, nil
}
