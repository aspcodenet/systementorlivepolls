package utils

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func RandString(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ContainsIgnoreCase(fullstring string, searchfor string) bool {
	return strings.Contains(
		strings.ToLower(fullstring),
		strings.ToLower(searchfor),
	)
}
