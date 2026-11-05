// Package redact masks credential values in YAML documents.
//
// Every consumer that prints or logs a cluster configuration shares this
// package so that they mask the same fields: the "describe config" report and
// the debug dump of a freshly loaded configuration. Redaction is driven by a
// known set of mapping keys and command line flag names, not by matching the
// shape of values, so a legitimate value is never corrupted and an unknown
// flag is left alone.
//
// Host connection credentials (for example the WinRM password) live on types
// owned by the rig dependency, which launchpad cannot give a masking
// MarshalYAML. Redacting the encoded document instead of the Go types is what
// lets those fields be covered at all.
package redact

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// Placeholder is substituted for every redacted value.
const Placeholder = "[REDACTED]"

// sensitiveKeys are YAML mapping keys whose value is a credential. Keys are
// compared case insensitively, which covers both the camelCase schema keys
// (adminPassword) and the lowercase rig connection keys (password).
//
// certData and keyData hold the client certificate and its private key as
// inline PEM. license holds MCR license content, which the installer writes
// to docker.lic at mode 0400 (pkg/configurer/linux.go). configData holds
// opaque blobs loaded from spec.mke.configFile and spec.mke.cloud.configFile
// by MKEConfig.UnmarshalYAML: an MKE config.toml carries LDAP bind
// credentials and a cloud provider config carries the provider's client
// secret, so the blob is masked whole rather than parsed. caCertData is
// deliberately absent: a CA bundle is public trust material and is useful
// when reading a dumped configuration.
var sensitiveKeys = map[string]struct{}{
	"password":      {},
	"adminpassword": {},
	"adminusername": {},
	"username":      {},
	"certdata":      {},
	"keydata":       {},
	"license":       {},
	"configdata":    {},
}

// sensitiveFlags are command line flags whose argument is a credential. MKE
// and MSR install/upgrade flags are free-form string lists, so a secret passed
// this way is part of a list element rather than a mapping value.
var sensitiveFlags = map[string]struct{}{
	"--admin-password": {},
	"--admin-username": {},
	"--ucp-password":   {},
	"--ucp-username":   {},
	"--password":       {},
	"--dtr-key":        {},
	"--token":          {},
}

// IsSensitiveKey reports whether a YAML mapping key holds a credential.
func IsSensitiveKey(key string) bool {
	_, found := sensitiveKeys[strings.ToLower(key)]

	return found
}

// IsSensitiveFlag reports whether a command line flag's argument is a credential.
func IsSensitiveFlag(flag string) bool {
	_, found := sensitiveFlags[strings.ToLower(flag)]

	return found
}

// YAML re-encodes a YAML mapping document with every credential value replaced
// by Placeholder. Mapping key order is preserved; comments and flow style are
// not, because the document is parsed and re-encoded.
func YAML(doc []byte) ([]byte, error) {
	var root yaml.MapSlice
	if err := yaml.Unmarshal(doc, &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML for redaction: %w", err)
	}

	out, err := yaml.Marshal(redactValue(root))
	if err != nil {
		return nil, fmt.Errorf("failed to encode redacted YAML: %w", err)
	}

	return out, nil
}

// redactValue walks a decoded document. Decoding into a yaml.MapSlice makes
// every nested mapping a yaml.MapSlice too, so these are the only container
// types that can appear.
func redactValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case yaml.MapSlice:
		return redactMapping(typed)
	case []interface{}:
		return redactSequence(typed)
	default:
		return value
	}
}

func redactMapping(mapping yaml.MapSlice) yaml.MapSlice {
	out := make(yaml.MapSlice, len(mapping))

	for i, item := range mapping {
		key, isString := item.Key.(string)
		if isString && IsSensitiveKey(key) && !isEmpty(item.Value) {
			out[i] = yaml.MapItem{Key: item.Key, Value: Placeholder}

			continue
		}

		out[i] = yaml.MapItem{Key: item.Key, Value: redactValue(item.Value)}
	}

	return out
}

// redactSequence masks flag arguments in a free-form flag list.
func redactSequence(seq []interface{}) []interface{} {
	out := make([]interface{}, len(seq))
	argIsSecret := false

	for i, item := range seq {
		str, isString := item.(string)
		if !isString {
			out[i] = redactValue(item)
			argIsSecret = false

			continue
		}

		out[i], argIsSecret = redactFlags(str, argIsSecret)
	}

	return out
}

// redactFlags masks credential arguments in one element of a flag list and
// reports whether the element ended on a bare credential flag, whose argument
// is then the next element.
//
// An element is not necessarily one flag. Flags.Join() concatenates the list
// into a single shell fragment, so "--ucp-insecure-tls --ucp-password=secret"
// in one element is both legal and effective; leading and repeated whitespace
// are equally legal. Every element is therefore split into tokens, and each
// token is matched by flag name. Untouched tokens keep their original spacing.
func redactFlags(element string, argIsSecret bool) (string, bool) {
	var out strings.Builder

	for _, tok := range splitTokens(element) {
		out.WriteString(tok.space)

		if tok.text == "" {
			continue
		}

		if argIsSecret {
			out.WriteString(Placeholder)
			argIsSecret = false

			continue
		}

		text, isFlagName := redactToken(tok.text)
		out.WriteString(text)
		argIsSecret = isFlagName
	}

	return out.String(), argIsSecret
}

// redactToken masks "--flag=value" and reports whether the token is a bare
// credential flag name, whose argument is the token that follows. A flag that
// is not known to carry a credential is returned untouched.
func redactToken(tok string) (string, bool) {
	idx := strings.IndexByte(tok, '=')
	if idx < 0 {
		return tok, IsSensitiveFlag(tok)
	}

	if !IsSensitiveFlag(tok[:idx]) {
		return tok, false
	}

	return tok[:idx] + "=" + Placeholder, false
}

// token is one whitespace-separated shell word plus the whitespace preceding
// it, so that an element can be rebuilt with its original spacing.
type token struct {
	space string
	text  string
}

// splitTokens splits a shell fragment on unquoted whitespace. Quoting matters:
// the repo writes --admin-password='...' (install_mke.go), and a password may
// contain spaces, which must not split the value into a separate token.
func splitTokens(element string) []token {
	var tokens []token

	for pos := 0; pos < len(element); {
		spaceStart := pos
		for pos < len(element) && isSpace(element[pos]) {
			pos++
		}

		textStart := pos

		var quote byte

		for pos < len(element) {
			char := element[pos]

			switch {
			case quote != 0:
				if char == quote {
					quote = 0
				}
			case char == '\'' || char == '"':
				quote = char
			case isSpace(char):
				quote = 0
			}

			if quote == 0 && isSpace(char) {
				break
			}

			pos++
		}

		tokens = append(tokens, token{space: element[spaceStart:textStart], text: element[textStart:pos]})
	}

	return tokens
}

func isSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	str, isString := value.(string)

	return isString && str == ""
}
