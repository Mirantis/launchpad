package test

import (
	"crypto/rand"
	"math/big"
)

// GenerateRandomAlphaNumericString generates a random string of a given length with only alphanumeric values.
// The values come from crypto/rand, because callers use this for cluster
// credentials; rand.Int over the charset length is rejection sampled by the
// standard library, so the characters are uniformly distributed.
func GenerateRandomAlphaNumericString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := make([]byte, length)

	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// crypto/rand.Reader does not fail on any platform this suite
			// builds for; a failure here means the system entropy source is
			// broken and generating a weaker credential instead is not an
			// acceptable fallback.
			panic("GenerateRandomAlphaNumericString: crypto/rand failed: " + err.Error())
		}

		result[i] = charset[n.Int64()]
	}

	return string(result)
}
