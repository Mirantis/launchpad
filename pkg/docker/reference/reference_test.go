package reference

import (
	"strings"
	"testing"
)

// hex64 is a 64-character hex string, the shape of a sha256 digest. Built
// rather than written out so it cannot be mistaken for a credential.
var hex64 = strings.Repeat("ab", 32)

func TestIsValidRepository(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Values launchpad itself ships or documents. These are the cases
		// that rule out reusing imageListFromOutputString, which requires a
		// tag and rejects every one of them.
		{name: "launchpad default repo", input: "docker.io/mirantis", want: true},
		{name: "launchpad legacy repo", input: "docker.io/docker", want: true},
		{name: "single component", input: "mirantis", want: true},
		{name: "registry with port", input: "registry.example.com:5000/mirantis", want: true},
		{name: "deep path", input: "registry.example.com/a/b/c", want: true},
		{name: "separators in a component", input: "docker.io/my-org.team_name", want: true},
		{name: "ipv6 registry", input: "[2001:db8::1]:5000/mirantis", want: true},

		// Shell metacharacters. Each would be inert today because every sink
		// quotes, but validation is the layer that does not depend on every
		// future sink remembering to.
		{name: "semicolon", input: "docker.io/mirantis;id", want: false},
		{name: "command substitution", input: "docker.io/$(id)", want: false},
		{name: "backtick", input: "docker.io/`id`", want: false},
		{name: "pipe", input: "docker.io/a|id", want: false},
		{name: "ampersand", input: "docker.io/a&id", want: false},
		{name: "redirect", input: "docker.io/a>/tmp/x", want: false},
		{name: "newline", input: "docker.io/a\nid", want: false},
		{name: "space", input: "docker.io/a b", want: false},
		{name: "single quote", input: "docker.io/a'b", want: false},
		{name: "double quote", input: "docker.io/a\"b", want: false},
		{name: "dollar brace", input: "docker.io/${HOME}", want: false},

		// Grammar violations that are not injection but are still malformed.
		{name: "empty", input: "", want: false},
		{name: "leading slash", input: "/mirantis", want: false},
		{name: "trailing slash", input: "docker.io/", want: false},
		{name: "double slash", input: "docker.io//mirantis", want: false},
		{name: "uppercase path component", input: "docker.io/Mirantis", want: false},
		{name: "parent traversal", input: "docker.io/../etc", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidRepository(tt.input); got != tt.want {
				t.Errorf("IsValidRepository(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidReference(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "repo with tag", input: "docker.io/mirantis/ucp:3.9.7", want: true},
		{name: "repo without tag", input: "docker.io/mirantis/ucp", want: true},
		{name: "tag with build metadata", input: "docker.io/mirantis/ucp:3.9.7-tp1", want: true},
		{name: "digest", input: "docker.io/mirantis/ucp@sha256:" + hex64, want: true},
		{name: "tag and digest", input: "docker.io/mirantis/ucp:3.9.7@sha256:" + hex64, want: true},

		{name: "metacharacter in tag", input: "docker.io/mirantis/ucp:3.9.7;id", want: false},
		{name: "command substitution in tag", input: "docker.io/mirantis/ucp:$(id)", want: false},
		{name: "space in tag", input: "docker.io/mirantis/ucp:3.9 7", want: false},
		{name: "empty tag", input: "docker.io/mirantis/ucp:", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidReference(tt.input); got != tt.want {
				t.Errorf("IsValidReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
