package redact

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ByLengthDesc orders redaction values longest first and drops empty ones.
//
// rig replaces each registered value in turn with strings.ReplaceAll, so a
// value that is a prefix of another destroys the longer match before it is
// tried: with the username "admin" and the password "admin!s3cr3t"
// registered in that order, the password is rewritten to "[REDACTED]!s3cr3t"
// and the distinctive part of it survives in the log. Registering the
// longest value first removes the dependence on registration order.
//
// An empty value is dropped because it would otherwise match at every
// position; rig skips empty matches too, but a caller should not rely on
// that.
func ByLengthDesc(values ...string) []string {
	ordered := make([]string, 0, len(values))

	for _, value := range values {
		if value != "" {
			ordered = append(ordered, value)
		}
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i]) > len(ordered[j])
	})

	return ordered
}

// fallbackPattern matches "<sensitive key><separator><value>" on one line,
// case insensitively, for every key in the sensitive set. It is built from
// the same key set as the structured redaction so the two cannot drift.
var fallbackPattern = buildFallbackPattern()

func buildFallbackPattern() *regexp.Regexp {
	keys := make([]string, 0, len(sensitiveKeys))
	for key := range sensitiveKeys {
		keys = append(keys, regexp.QuoteMeta(key))
	}

	sort.Strings(keys)

	// Also catch the environment-variable spellings, which are matched by
	// fragment in the structured path and cannot be enumerated here.
	fragments := make([]string, 0, len(envSecretFragments))
	for _, fragment := range envSecretFragments {
		fragments = append(fragments, `[A-Za-z0-9_]*`+regexp.QuoteMeta(fragment)+`[A-Za-z0-9_]*`)
	}

	return regexp.MustCompile(`(?im)^(\s*-?\s*(?:` + strings.Join(keys, "|") + "|" + strings.Join(fragments, "|") + `)\s*[:=]\s*).+$`)
}

// Text masks credentials in a document that could not be parsed.
//
// This is a best-effort fallback for YAML that fails to load: it is line
// oriented and key driven, so it cannot see a credential that is not on the
// same line as its key, and it masks whatever follows the separator without
// understanding the value. Callers that have a parseable document must use
// YAML, which walks the structure and knows where the credential-bearing
// keys actually are.
//
// It covers the same key set and the same environment-variable fragments as
// the structured path, case insensitively, so a configuration that fails to
// parse does not silently lose coverage the parsed one has.
func Text(doc []byte) []byte {
	return fallbackPattern.ReplaceAll(doc, []byte(fmt.Sprintf("${1}%s", Placeholder)))
}
