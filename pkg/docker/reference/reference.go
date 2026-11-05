// Package reference validates Docker/OCI image references against the
// distribution reference grammar.
//
// It exists separately from pkg/docker so that configuration packages can
// validate user-supplied values without importing pkg/docker, which imports
// the MKE config package and would create a cycle.
//
// The grammar here is NOT the regex at pkg/docker/image.go
// (imageListFromOutputString). That one scrapes image names out of
// bootstrapper stdout: it is line-anchored, and it requires both a slash and
// a tag. Applied to a repository it rejects "docker.io/mirantis", the value
// launchpad ships as its own default, so it cannot serve as a config
// validator. It is left doing the job it was written for.
package reference

import "regexp"

// Grammar, following github.com/distribution/reference:
//
//	name      := [domain '/'] path-component ['/' path-component]*
//	domain    := host [':' port-number]
//	host      := domain-name | IPv4 | '[' IPv6 ']'
//	component := alpha-numeric [separator alpha-numeric]*
//	separator := '.' | '_' | '__' | '-'+
//	tag       := [\w][\w.-]{0,127}
//	digest    := algorithm ':' hex
//
// Every part is anchored and no part admits a shell metacharacter: the
// character classes are alphanumerics plus '.', '_', '-', '/', ':' and the
// brackets of an IPv6 literal. That is what makes this an allowlist rather
// than a denylist -- a value nobody anticipated is rejected by default
// instead of being permitted because no rule named it.
const (
	alphaNumeric = `[a-z0-9]+`
	separator    = `(?:[._]|__|[-]+)`
	component    = alphaNumeric + `(?:` + separator + alphaNumeric + `)*`

	domainName = `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
	ipv6       = `\[(?:[a-fA-F0-9:]+)\]`
	host       = `(?:` + domainName + `(?:\.` + domainName + `)*|` + ipv6 + `)`
	domain     = host + `(?::[0-9]+)?`

	namePattern = `(?:` + domain + `/)?` + component + `(?:/` + component + `)*`
	tagPattern  = `[\w][\w.-]{0,127}`
	digest      = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*:[0-9a-fA-F]{32,}`
)

var (
	// nameRe matches a repository name: an optional registry domain followed
	// by one or more path components. This is the shape of spec.*.imageRepo.
	nameRe = regexp.MustCompile(`^` + namePattern + `$`)

	// referenceRe matches a full reference: a name with an optional tag, an
	// optional digest, or both.
	referenceRe = regexp.MustCompile(`^` + namePattern + `(?::` + tagPattern + `)?(?:@` + digest + `)?$`)
)

// IsValidRepository reports whether s is a well-formed repository name, with
// or without a registry domain. An empty string is not valid; callers that
// treat empty as "use the default" must apply the default first.
func IsValidRepository(s string) bool {
	return s != "" && len(s) <= maxLength && nameRe.MatchString(s)
}

// IsValidReference reports whether s is a well-formed image reference,
// optionally carrying a tag and/or a digest.
func IsValidReference(s string) bool {
	return s != "" && len(s) <= maxLength && referenceRe.MatchString(s)
}

// maxLength bounds the input so a pathological value cannot be handed to the
// regex engine, and reflects that no real reference approaches it.
const maxLength = 4096
