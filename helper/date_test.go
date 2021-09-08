package helper

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDateParse(t *testing.T) {
	// 2006-01-02T15:04:05.999999999Z07:00
	tests := []struct {
		value    string
		expected time.Time
	}{
		{
			value:    "2021-08-23T14:32:10+09:00",
			expected: time.Date(2021, 8, 23, 14, 32, 10, 0, jst),
		},
		{
			value:    "2021-08-23T14:32:10Z",
			expected: time.Date(2021, 8, 23, 14, 32, 10, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		tm, err := ParseISO8601(tt.value)
		if assert.NoError(t, err) {
			logrus.Info(tt.expected)
			logrus.Info(tm)
			assert.Equal(t, tt.expected.UnixNano(), tm.UnixNano())
		}
	}
}
