package helper

import "time"

var (
	jst = time.FixedZone("JST", 9*60*60)
)

const (
	ISO8601 = time.RFC3339Nano // "2006-01-02T15:04:05.9999999Z07:00"
)

func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(ISO8601, s)
}

func ParseISO8601InLocation(s string) (time.Time, error) {
	return time.ParseInLocation(ISO8601, s, jst)
}

func FormatISO8601(t time.Time) string {
	return t.Format(ISO8601)
}
