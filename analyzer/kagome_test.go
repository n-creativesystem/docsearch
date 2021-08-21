package analyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserDict(t *testing.T) {
	reader := strings.NewReader(`朝青龍,朝青龍,アサショウリュウ,カスタム人名`)
	expecteds := []string{
		"第",
		"68",
		"代",
		"横綱",
		"朝青龍",
	}
	query := []byte("第68代横綱朝青龍")
	d, err := UserDictionary(reader)
	if assert.NoError(t, err) {
		ana := d()
		stream := ana.Analyze(query)
		assert.Len(t, stream, len(expecteds))
		for i, token := range stream {
			assert.Equal(t, expecteds[i], string(token.Term))
		}
	}
}
