package byteutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 bytes"},
		{1, "1 bytes"},
		{1024, "1 KiB"},
		{1024*1024 - 1, "1023 KiB"},
		{1024 * 1024, "1 MiB"},
		{1024*1024*1024 - 1, "1023 MiB"},
		{1024 * 1024 * 1024, "1 GiB"},
		{1024*1024*1024*1024 - 1, "1023 GiB"},
		{1024 * 1024 * 1024 * 1024, "1 TiB"},
		{1024*1024*1024*1024*1024 - 1, "1023 TiB"},
	}
	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			actual := FormatBytes(tc.bytes)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
