package cmd

import (
	"os"
	"strings"
)

func getEnv(key string) string {
	return os.Getenv(key)
}

func hasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}
