package mkex

import (
	"errors"
	"fmt"

	"github.com/k0sproject/rig"
)

const (
	mkexDetectExtraFieldKey = "MKEX"
	mkexDetectReleaseFile   = "/usr/lib/mirantis-release"
)

var (
	errAbort    = errors.New("base os detected but version resolving failed") // duplicate from k0sproject/rig(resolver.go)
	ErrNotMirOS = fmt.Errorf("%w; not Mirantis MKEx OS", rig.ErrNotSupported)
)

func init() {
	// Add our MKEx OSVersion resolver to the rig set (as the first resolver)
	rig.Resolvers = append([]rig.ResolveFunc{mkexRigResolver}, rig.Resolvers...)
}

func isMKExOSVersion(v rig.OSVersion) bool {
	_, ok := v.ExtraFields[mkexDetectExtraFieldKey]
	return ok
}

// mkexRigResolver Resolve if this connection is for an MKEX host
//
//	if the MKEX file is on the machine, then it is an MKEX machine, and
//	so we treat it like a linux machine, but add our custom flag.
func mkexRigResolver(conn *rig.Connection) (rig.OSVersion, error) {
	if conn.IsWindows() {
		return rig.OSVersion{}, ErrNotMirOS
	}
	output, err := conn.ExecOutput(fmt.Sprintf("cat %s", mkexDetectReleaseFile))
	if err != nil {
		return rig.OSVersion{}, fmt.Errorf("%w: %s", ErrNotMirOS, err.Error())
	}

	osv, err := resolveLinux(conn)
	if err != nil {
		return osv, fmt.Errorf("%w: %s", ErrNotMirOS, err.Error())
	}

	osv.ExtraFields[mkexDetectExtraFieldKey] = output
	osv.Name = fmt.Sprintf("%s [%s]", osv.Name, output) // without this it is hard to see in the output that we did resolve.
	return osv, nil
}

// resolveLinux duplicated from k0sprojest/rig(resolver.go).
func resolveLinux(conn *rig.Connection) (rig.OSVersion, error) {
	if err := conn.Exec("uname | grep -q Linux"); err != nil {
		return rig.OSVersion{}, fmt.Errorf("not a linux host (%w)", err)
	}

	output, err := conn.ExecOutput("cat /etc/os-release || cat /usr/lib/os-release")
	if err != nil {
		// at this point it is known that this is a linux host, so any error from here on should signal the resolver to not try the next
		return rig.OSVersion{}, fmt.Errorf("%w: unable to read os-release file: %w", errAbort, err)
	}

	var version rig.OSVersion
	if err := rig.ParseOSReleaseFile(output, &version); err != nil {
		return rig.OSVersion{}, errors.Join(errAbort, err)
	}
	return version, nil
}
