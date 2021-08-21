package model_test

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/blugelabs/bluge"
	"github.com/n-creativesystem/docsearch/helper"
	"github.com/n-creativesystem/docsearch/model"
	"github.com/stretchr/testify/assert"
)

func TestFieldMapping(t *testing.T) {
	expectedTime, _ := helper.ParseISO8601("2021-08-17T12:30:00+09:00")
	tests := []struct {
		name     string
		value    string
		excepted bluge.Field
	}{
		{
			name:     "test1",
			value:    `{"value": "text", "type": "keyword"}`,
			excepted: bluge.NewKeywordField("test", "text").StoreValue(),
		},
		{
			name:     "test2",
			value:    `{"value": "text body", "type": "text"}`,
			excepted: bluge.NewTextField("test", "text body").StoreValue(),
		},
		{
			name:     "test3",
			value:    `{"value": 1, "type": "numeric"}`,
			excepted: bluge.NewNumericField("test", 1).StoreValue(),
		},
		{
			name:     "test4",
			value:    `{"value": 1.1, "type": "numeric"}`,
			excepted: bluge.NewNumericField("test", 1.1).StoreValue(),
		},
		{
			name:     "test5",
			value:    fmt.Sprintf(`{"value": %f, "type": "numeric"}`, math.MaxFloat32),
			excepted: bluge.NewNumericField("test", math.MaxFloat32).StoreValue(),
		},
		{
			name:     "test6",
			value:    fmt.Sprintf(`{"value": %f, "type": "numeric"}`, math.MaxFloat64),
			excepted: bluge.NewNumericField("test", math.MaxFloat64).StoreValue(),
		},
		{
			name:     "test7",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxInt16),
			excepted: bluge.NewNumericField("test", math.MaxInt16).StoreValue(),
		},
		{
			name:     "test8",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxInt32),
			excepted: bluge.NewNumericField("test", math.MaxInt32).StoreValue(),
		},
		{
			name:     "test9",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxInt64),
			excepted: bluge.NewNumericField("test", math.MaxInt64).StoreValue(),
		},
		{
			name:     "test10",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxInt8),
			excepted: bluge.NewNumericField("test", math.MaxInt8).StoreValue(),
		},
		{
			name:     "test11",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxUint16),
			excepted: bluge.NewNumericField("test", math.MaxUint16).StoreValue(),
		},
		{
			name:     "test12",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxUint32),
			excepted: bluge.NewNumericField("test", math.MaxUint32).StoreValue(),
		},
		{
			name:     "test13",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, uint64(math.MaxUint64)),
			excepted: bluge.NewNumericField("test", math.MaxUint64).StoreValue(),
		},
		{
			name:     "test14",
			value:    fmt.Sprintf(`{"value": %d, "type": "numeric"}`, math.MaxUint8),
			excepted: bluge.NewNumericField("test", math.MaxUint8).StoreValue(),
		},
		{
			name:     "test15",
			value:    fmt.Sprintf(`{"value": {"lat": %.10f, "lon": %.10f}, "type": "geopoint"}`, 35.65809922, 139.74135249),
			excepted: bluge.NewGeoPointField("test", 139.74135249, 35.65809922).StoreValue(),
		},
		{
			name:     "test16",
			value:    `{"value": "2021-08-17T12:30:00+09:00", "type": "datetime"}`,
			excepted: bluge.NewDateTimeField("test", expectedTime).StoreValue(),
		},
	}

	for _, tt := range tests {
		var fieldMapping model.FieldMapping
		err := json.Unmarshal([]byte(tt.value), &fieldMapping)
		if assert.NoError(t, err) {
			f, err := fieldMapping.BlugeField("test")
			if assert.NoError(t, err) {
				if !reflect.DeepEqual(f, tt.excepted) {
					t.Errorf("Test name: %s, Expected %#v, value: %s", tt.name, f, string(tt.excepted.Value()))
				}
			}
		}
	}
}
