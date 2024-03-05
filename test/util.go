package test

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateRandomString generates a random string of a given length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
