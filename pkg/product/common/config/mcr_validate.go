package config

import (
	"net/url"
	"regexp"
	"strings"
)

// The patterns below are derived from where the values are interpolated, not
// from the values Mirantis happens to publish today. A pattern justified by
// "these are the channels I have seen" is a guess; a pattern justified by
// "this is what the sinks can carry" survives a new channel being published.
//
// channel is interpolated into:
//
//   - a yum .repo baseurl path                (enterpriselinux/el.go)
//   - a zypper repo URL path                  (sles/sles.go)
//   - the Components: field of an apt deb822 source, which is
//     whitespace-delimited                    (ubuntu/ubuntu.go)
//   - a -Channel parameter to the Windows installer (configurer/windows.go)
//
// A channel is one or two path segments. Two, because the FIPS channels are
// published as a subdirectory and not as a decoration of the name:
// repos.mirantis.com/rhel/9/x86_64/stable-29.2.1/fips/ is a real path, and
// test/smoke/smoke_test.go and PRODENG-4062's installer test both use that
// form. An earlier revision of this validator allowed one segment only, on
// the reasoning that a "/" would change the path depth -- which is true, and
// is exactly what the FIPS channel deliberately does.
//
// What the sinks still forbid: whitespace, which would split the apt
// component list; "?" and "#", which would truncate the URL; a leading or
// trailing "/", an empty segment, and the dot-names, which would change
// which directory is addressed rather than extend the path.
//
// If Mirantis ever publishes a channel this rejects, the constraint is what
// needs revisiting -- not the channel -- and the reader can see why from this
// comment rather than guessing whether the restriction was deliberate.
const mcrChannelSegment = `[a-zA-Z0-9][a-zA-Z0-9._-]*`

var mcrChannelRe = regexp.MustCompile(`^` + mcrChannelSegment + `(?:/` + mcrChannelSegment + `)?$`)

// mcrRepoURLSchemes are the schemes the package managers this value is handed
// to can actually fetch. file is included deliberately: an air-gapped install
// pointing at a locally mounted mirror is legitimate, and yum, zypper and apt
// all accept it. This is a different question from the installer script URL,
// which launchpad fetches itself and which PRODENG-3924 restricts to https --
// that value is retrieved by launchpad over the network, this one is handed
// to the host's package manager.
var mcrRepoURLSchemes = map[string]bool{"http": true, "https": true, "file": true}

// mcrRepoURLCharsRe restricts the URL to characters RFC 3986 permits. It is
// deliberately not a shell-metacharacter denylist: "$", "(" and ")" are legal
// in a URL and are permitted here. Quoting at the command sites is what makes
// them safe, and validation that duplicated the quoter's job would give a
// false impression of where the protection lives.
var mcrRepoURLCharsRe = regexp.MustCompile(`^[A-Za-z0-9._~:/?#\[\]@!$&'()*+,;=%-]+$`)

// IsValidMCRChannel reports whether s is usable as an MCR channel at every
// site it reaches.
func IsValidMCRChannel(s string) bool {
	if s == "" || len(s) > 256 || !mcrChannelRe.MatchString(s) {
		return false
	}

	// The segment pattern cannot produce "." or ".." on its own, but spelling
	// the check out keeps the traversal guarantee visible at the point a
	// reader looks for it rather than requiring them to re-derive it.
	for _, seg := range strings.Split(s, "/") {
		if seg == "." || seg == ".." {
			return false
		}
	}

	return true
}

// IsValidMCRRepoURL reports whether s is an absolute URL with a scheme the
// package managers can fetch.
func IsValidMCRRepoURL(s string) bool {
	if s == "" || len(s) > 2048 || !mcrRepoURLCharsRe.MatchString(s) {
		return false
	}

	parsed, err := url.Parse(s)
	if err != nil || !parsed.IsAbs() || !mcrRepoURLSchemes[parsed.Scheme] {
		return false
	}

	// file:// has no host; everything else must name one, or the rendered
	// repo URL points nowhere.
	return parsed.Scheme == "file" || parsed.Host != ""
}
