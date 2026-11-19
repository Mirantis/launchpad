//go:build testing

package shellword_test

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/util/shellword"
	"github.com/stretchr/testify/require"
)

func TestUnquote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bare", "simple", "simple"},
		{"empty", "", ""},
		{"empty single quotes", "''", ""},
		{"empty double quotes", `""`, ""},
		{"single quoted", "'p@ss word'", "p@ss word"},
		{"double quoted", `"p@ss word"`, "p@ss word"},
		{"backslash is literal inside double quotes", `"C:\temp"`, `C:\temp`},
		{"backslash escapes a double quote", `"a\"b"`, `a"b`},
		{"backslash escapes a dollar", `"pa\$word"`, "pa$word"},
		{"backslash escapes a backslash", `"a\\b"`, `a\b`},
		{"backslash is literal inside single quotes", `'a\b'`, `a\b`},
		{"bare backslash escapes the next byte", `a\b`, "ab"},
		{"concatenated sections", `'it'"'"'s'`, "it's"},
		{"hash", "'a#b'", "a#b"},
		{"double quote inside single quotes", `'a"b'`, `a"b`},
		{"unterminated single quote", "'abc", "abc"},
		{"unterminated double quote", `"abc`, "abc"},
		{"trailing backslash is dropped", `abc\`, "abc"},
		{"command substitution is not expanded", `"$(id)"`, "$(id)"},
		{"backtick is not expanded", "'`id`'", "`id`"},
		{"variable is not expanded", `"$HOME"`, "$HOME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, shellword.Unquote(tt.in))
		})
	}
}
