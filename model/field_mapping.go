package model

import (
	"github.com/blugelabs/bluge"
	"github.com/n-creativesystem/docsearch/analyzer"
	"github.com/n-creativesystem/docsearch/errors"
	"github.com/n-creativesystem/docsearch/helper"
)

type FieldMapping struct {
	Value            interface{} `json:"value"`
	AnalyserName     string      `json:"analyzer_name"`
	Type             MappingType `json:"type"`
	StoreValue       bool        `json:"store_value"`
	HighlightMatches bool        `json:"highlight_matches"`
}

func (f *FieldMapping) SetAnalyzer(field *bluge.TermField) *bluge.TermField {
	analyzer := analyzer.GetAnalyzer(f.AnalyserName)
	if analyzer != nil {
		field = field.WithAnalyzer(analyzer)
	}
	return field
}

func (f *FieldMapping) BlugeField(name string) (bluge.Field, error) {
	switch f.Type {
	case Keyword:
		if v, ok := f.Value.(string); ok {
			field := bluge.NewKeywordField(name, v)
			if f.StoreValue {
				field.StoreValue()
			}
			if f.HighlightMatches {
				field.HighlightMatches()
			}
			return f.SetAnalyzer(field), nil
		} else {
			return nil, errors.New("keyword type is a string")
		}
	case Numeric:
		if v, ok := f.Value.(float64); ok {
			field := bluge.NewNumericField(name, v)
			if f.StoreValue {
				field.StoreValue()
			}
			if f.HighlightMatches {
				field.HighlightMatches()
			}
			return f.SetAnalyzer(field), nil
		}
		return nil, errors.New("numeric type is numeric")
	case Geopoint:
		if v, ok := f.Value.(map[string]interface{}); ok {
			lon, lonOk := v["lon"].(float64)
			lat, latOk := v["lat"].(float64)
			if !lonOk {
				lon, lonOk = v["lng"].(float64)
			}
			if latOk && lonOk {
				field := bluge.NewGeoPointField(name, lon, lat)
				if f.StoreValue {
					field.StoreValue()
				}
				if f.HighlightMatches {
					field.HighlightMatches()
				}
				return f.SetAnalyzer(field), nil
			}
			return nil, errors.New("geopoint type is a {\"lon\": float,\"lat\": float}")
		}
	case Text:
		switch v := f.Value.(type) {
		case string:
			field := bluge.NewTextField(name, v)
			if f.StoreValue {
				field.StoreValue()
			}
			if f.HighlightMatches {
				field.HighlightMatches()
			}
			return f.SetAnalyzer(field), nil
		case []byte:
			field := bluge.NewTextFieldBytes(name, v)
			if f.StoreValue {
				field.StoreValue()
			}
			if f.HighlightMatches {
				field.HighlightMatches()
			}
			return f.SetAnalyzer(field), nil
		default:
			return nil, errors.New("text type is a string or byte array")
		}
	case DateTime:
		if v, ok := f.Value.(string); ok {
			dt, err := helper.ParseISO8601(v)
			if err != nil {
				return nil, errors.New("datetime parse error: %s", v)
			}
			field := bluge.NewDateTimeField(name, dt)
			if f.StoreValue {
				field.StoreValue()
			}
			if f.HighlightMatches {
				field.HighlightMatches()
			}
			return f.SetAnalyzer(field), nil
		}
	}
	return nil, errors.New("No support type")
}
