package config

import (
	"net/url"
	"regexp"
)

// The patterns below are derived from where the values are interpolated, not
// from the values Mirantis happens to publish today. A pattern justified by
// "these are the channels I have seen" is a guess; a pattern justified by
// "this is what the sinks can carry" survives a new channel being published.
//
// channel is interpolated into:
//
//   - a yum .repo baseurl path segment   (enterpriselinux/el.go)
//   - a zypper repo URL path segment     (sles/sles.go)
//   - the Components: field of an apt deb822 source, which is
//     whitespace-delimited (ubuntu/ubuntu.go)
//   - a -Channel parameter to the Windows installer (configurer/windows.go)
//
// The union of those requires a single URL path segment that is also a single
// whitespace-delimited token: no "/", which would change the path depth; no
// "?" or "#", which would truncate the URL; no whitespace, which would split
// the apt component list.
//
// If Mirantis ever publishes a channel this rejects, the constraint is what
// needs revisiting -- not the channel -- and the reader can see why from this
// comment rather than guessing whether the restriction was deliberate.
var mcrChannelRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

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
	return s != "" && len(s) <= 256 && mcrChannelRe.MatchString(s)
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
