package helper

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	jst = time.FixedZone("Asia/Tokyo", 9*60*60)
)

const (
	ISO8601 = time.RFC3339Nano
)

// Str2bytes converts string("00 00 00 00 00 00 00 00") to []byte
func Str2bytes(str string) []byte {
	bytes := make([]byte, 8)
	for i, e := range strings.Fields(str) {
		b, _ := strconv.ParseUint(e, 16, 64)
		bytes[i] = byte(b)
	}
	return bytes
}

// Bytes2str converts []byte to string("00 00 00 00 00 00 00 00")
func Bytes2str(bytes ...byte) string {
	strs := []string{}
	for _, b := range bytes {
		strs = append(strs, fmt.Sprintf("%02x", b))
	}
	return strings.Join(strs, " ")
}

// Bytes2uint converts []byte to uint64
func Bytes2uint(bytes ...byte) uint64 {
	padding := make([]byte, 8-len(bytes))
	i := binary.BigEndian.Uint64(append(padding, bytes...))
	return i
}

// Uint2bytes converts uint64 to []byte
func Uint2bytes(i uint64, size int) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, i)
	return bytes[8-size : 8]
}

// Bytes2int converts []byte to int64
func Bytes2int(bytes ...byte) int64 {
	if 0x7f < bytes[0] {
		mask := uint64(1<<uint(len(bytes)*8-1) - 1)

		bytes[0] &= 0x7f
		i := Bytes2uint(bytes...)
		i = (^i + 1) & mask
		return int64(-i)

	} else {
		i := Bytes2uint(bytes...)
		return int64(i)
	}
}

// Int2bytes converts int to []byte
func Int2bytes(i int, size int) []byte {
	var ui uint64
	if 0 < i {
		ui = uint64(i)
	} else {
		ui = (^uint64(-i) + 1)
	}
	return Uint2bytes(ui, size)
}

func Interface2Float64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case uint:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	}
	return 0, false
}

func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(ISO8601, s)
}

func ParseISO8601InLocation(s string) (time.Time, error) {
	return time.ParseInLocation(ISO8601, s, jst)
}

func FormatISO8601(t time.Time) string {
	return t.Format(ISO8601)
}

func GetFileNameWithoutExt(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}
