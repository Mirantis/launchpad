package enterpriselinux

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/configurer"
)

// Rhel RedHat Enterprise Linux
type Rhel struct {
	Configurer
}

func resolveRedhatConfigurer(h *config.Host) config.HostConfigurer {
	if h.Metadata.Os.ID == "rhel" {
		return &Rhel{
			Configurer: Configurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
	}

	return nil
}

func init() {
	config.RegisterHostConfigurer(resolveRedhatConfigurer)
}
