package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
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
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "26.04"
		},
		func() interface{} {
			return ResoluteConfigurer{}
		},
	)
}
