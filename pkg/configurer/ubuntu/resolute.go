package ubuntu

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// ResoluteConfigurer is the Ubuntu Resolute Raccoon (26.04) specific host configurer implementation.
//
// NOTE: MCR has not published Docker EE packages for Ubuntu 26.04 as of
// 2026-09-02 (repos.mirantis.com/ubuntu/dists/ and
// repos-internal.mirantis.com/ubuntu/dists/ only have
// trusty/xenial/bionic/focal/jammy/noble - no resolute). InstallMCR will
// fail on this platform until MCR ships support; see PRODENG-3593 and the
// TestCuttingEdgeCluster smoke test.
type ResoluteConfigurer struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ubuntu" && os.Version == "26.04"
		},
		func() interface{} {
			return ResoluteConfigurer{}
		},
	)
}
