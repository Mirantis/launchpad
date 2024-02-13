package version

import (
	"strings"

	"github.com/hashicorp/go-version"
)

func tp2qp(s string) string {
	return strings.Replace(s, "-tp", "-qp", 1)
}

// GreaterThan is a "corrected" version comparator that considers -tpX releases to be earlier than -rcX.
func GreaterThan(a, b *version.Version) bool {
	ca, _ := version.NewVersion(tp2qp(a.String()))
	cb, _ := version.NewVersion(tp2qp(b.String()))
	return ca.GreaterThan(cb)
}
