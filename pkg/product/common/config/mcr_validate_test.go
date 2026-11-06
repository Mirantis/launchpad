package config

import "testing"

func TestIsValidMCRChannel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Values Mirantis publishes. These pass as a consequence of the
		// sink-derived rule, not as its source.
		{name: "stable", input: "stable", want: true},
		{name: "test", input: "test", want: true},
		{name: "versioned stable", input: "stable-29.8.0", want: true},
		{name: "versioned test", input: "test-29.8.0", want: true},
		{name: "underscore", input: "stable_29", want: true},
		// FIPS channels are published as a subdirectory, not as a name
		// decoration: repos.mirantis.com/rhel/9/x86_64/stable-29.2.1/fips/
		// is a real path. test/smoke/smoke_test.go and PRODENG-4062's
		// installer test both use this form, and an earlier revision of
		// this validator rejected it.
		{name: "fips channel", input: "stable-29.2.1/fips", want: true},
		{name: "fips test channel", input: "test-29.9.2/fips", want: true},

		// Would break the apt Components: field, which is whitespace-delimited.
		{name: "space", input: "stable test", want: false},
		{name: "tab", input: "stable\ttest", want: false},
		{name: "newline", input: "stable\nSuites: evil", want: false},

		// Would change the URL path depth or truncate it.
		{name: "traversal via a second segment", input: "stable/../gpg", want: false},
		{name: "three segments", input: "stable/fips/extra", want: false},
		{name: "trailing slash", input: "stable/", want: false},
		{name: "leading slash", input: "/stable", want: false},
		{name: "empty second segment", input: "stable//fips", want: false},
		{name: "dot segment", input: "stable/.", want: false},
		{name: "dotdot segment", input: "stable/..", want: false},
		{name: "query", input: "stable?a=b", want: false},
		{name: "fragment", input: "stable#frag", want: false},

		// cmd.exe / shell metacharacters. Quoting is the control; rejecting
		// them here makes the bad value visible earlier.
		{name: "ampersand chain", input: `stable" && whoami && "`, want: false},
		{name: "pipe", input: "stable|whoami", want: false},
		{name: "command substitution", input: "stable$(id)", want: false},
		{name: "caret", input: "stable^", want: false},

		{name: "empty", input: "", want: false},
		{name: "leading dash", input: "-stable", want: false},
		{name: "leading dot", input: ".stable", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMCRChannel(tt.input); got != tt.want {
				t.Errorf("IsValidMCRChannel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidMCRRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "the shipped default", input: "https://repos.mirantis.com", want: true},
		{name: "http mirror", input: "http://mirror.internal.example.com", want: true},
		{name: "mirror with port and path", input: "https://mirror.example.com:8443/mcr", want: true},
		// Air-gapped installs pointing at a mounted mirror are legitimate,
		// and yum, zypper and apt all accept file.
		{name: "file mirror", input: "file:///srv/mirror/mcr", want: true},

		{name: "no scheme", input: "repos.mirantis.com", want: false},
		{name: "scheme the package managers cannot fetch", input: "ssh://repos.mirantis.com", want: false},
		{name: "javascript scheme", input: "javascript:alert(1)", want: false},
		{name: "http without host", input: "http://", want: false},
		{name: "space", input: "https://repos.mirantis.com/a b", want: false},
		{name: "newline injecting a repo directive", input: "https://repos.mirantis.com\nbaseurl=evil", want: false},
		{name: "backtick", input: "https://repos.mirantis.com/`id`", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMCRRepoURL(tt.input); got != tt.want {
				t.Errorf("IsValidMCRRepoURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
