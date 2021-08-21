package model

import (
	"encoding/json"
	"fmt"
)

type MappingType int

const (
	Unknown MappingType = iota
	Text
	Keyword
	Numeric
	NumericRange
	Geopoint
	DateTime

	text         = "text"
	keyword      = "keyword"
	numeric      = "numeric"
	numericRange = "numeric_range"
	geopoint     = "geopoint"
	dateTime     = "datetime"
)

var (
	mappingTypeOfString = map[string]MappingType{
		text:         Text,
		keyword:      Keyword,
		numeric:      Numeric,
		numericRange: NumericRange,
		geopoint:     Geopoint,
		dateTime:     DateTime,
	}
	stringOfMappingType = map[MappingType]string{
		Text:         text,
		Keyword:      keyword,
		Numeric:      numeric,
		NumericRange: numericRange,
		Geopoint:     geopoint,
		DateTime:     dateTime,
	}
)

func (m MappingType) String() string {
	if v, ok := stringOfMappingType[m]; ok {
		return v
	}
	return text
}

func (m MappingType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *MappingType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("data should be a string, got %s", data)
	}
	mt, ok := mappingTypeOfString[s]
	if !ok {
		*m = Unknown
	} else {
		*m = mt
	}
	return nil
}
