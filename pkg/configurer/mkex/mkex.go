package mkex

// it is important that the el.Rocky.init() is run before our init()
// so that our registry.RegisterOSModule() call is run second, giving
// our register priority.
import (
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/registry"
	log "github.com/sirupsen/logrus"
)

type RockyLinux struct {
	enterpriselinux.RockyLinux
}

func init() {
	registry.RegisterOSModule(
		isMKExOSVersion,
		func() interface{} {
			log.Warn("Building an MKEX configurer for a host. This configurer will behave differently, skipping some expected installation steps.")
			return RockyLinux{}
		},
	)
}

// InstallMKEBasePackages install all the needed base packages on the host.
func (c RockyLinux) InstallMKEBasePackages(h os.Host) error {
	log.Debugf("%s: InstallMKEBasePackage on MKEx takes no action. The base OS is expected to meet requirements already.", h)
	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c RockyLinux) UninstallMCR(h os.Host, _ string, _ common.MCRConfig) error {
	log.Debugf("%s: UninstallMCR on MKEx takes no action. The base OS should not be managed by Launchpad.", h)
	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c RockyLinux) InstallMCR(h os.Host, _ string, _ common.MCRConfig) error {
	log.Debugf("%s: InstallMCR on MKEx takes no action. The base OS is expected to meet requirements already", h)
	return nil
}

// AuthorizeDocker adds the current user to the docker group.
func (c RockyLinux) AuthorizeDocker(h os.Host) error {
	log.Debugf("%s: AuthorizeDocker on MKEx takes no action. The base OS is expected to have docker user/groups setup.", h)
	return nil
}

// CleanupLingeringMCR removes left over MCR files after Launchpad reset.
func (c RockyLinux) CleanupLingeringMCR(h os.Host, _ common.DockerInfo) {
	log.Debugf("%s: UninstallMCR on MKEx takes no action. The base OS should not be managed by Launchpad", h)
}
