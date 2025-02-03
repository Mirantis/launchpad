package phase

import (
	"fmt"

	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/centos"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/mkex"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/oracle"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/sles"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/ubuntu"
	// anonymous import is needed to load the os configurers.
	_ "github.com/Mirantis/launchpad/pkg/configurer/windows"
	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// DetectOS performs remote OS detection.
type DetectOS struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *DetectOS) Title() string {
	return "Detect host operating systems"
}

// Run the phase.
func (p *DetectOS) Run() error {
	err := p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if err := h.ResolveConfigurer(); err != nil {
			return fmt.Errorf("failed to resolve configurer for %s: %w", h, err)
		}
		os := h.OSVersion.String()
		log.Infof("%s: is running %s", h, os)

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to detect OS: %w", err)
	}
	return nil
}
