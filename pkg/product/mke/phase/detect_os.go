package phase

import (
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/oracle"
	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/sles"
	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// anonymous import is needed to load the os configurers
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"

	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// DetectOS performs remote OS detection
type DetectOS struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *DetectOS) Title() string {
	return "Detect host operating systems"
}

// Run the phase
func (p *DetectOS) Run() error {
	return p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if err := h.ResolveConfigurer(); err != nil {
			return err
		}
		os := h.OSVersion.String()
		log.Infof("%s: is running %s", h, os)

		return nil
	})
}
