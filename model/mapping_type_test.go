package model_test

import (
	"encoding/json"
	"testing"

	"github.com/n-creativesystem/docsearch/model"
	"github.com/stretchr/testify/assert"
)

func TestMappingType(t *testing.T) {
	type testStruct struct {
		Type model.MappingType `json:"type"`
	}
	var (
		expectedStr    = `{"type":"numeric"}`
		expectedStruct testStruct
	)

	testStr := testStruct{
		Type: model.Numeric,
	}
	data, err := json.Marshal(&testStr)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, expectedStr, string(data))
	err = json.Unmarshal(data, &expectedStruct)
	assert.NoError(t, err)
	assert.Equal(t, expectedStruct.Type, model.Numeric)
}
