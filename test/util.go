package test

import (
	"math/rand"
)

// GenerateRandomAlphaNumericString generates a random string of a given length with only alphanumeric values.
func GenerateRandomAlphaNumericString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		//nolint:gosec
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
