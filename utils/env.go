package utils

import "os"

func DefaultGetEnv(key, default_ string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return default_
}
