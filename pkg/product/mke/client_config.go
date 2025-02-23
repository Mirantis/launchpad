package mke

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	de "github.com/Mirantis/launchpad/pkg/product/mke/phase"
)

// ClientConfig downloads MKE client bundle.
func (p *MKE) ClientConfig() error {
	manager := p.ClusterConfig.Spec.Managers()[0]
	newHosts := make(api.Hosts, 1)
	newHosts[0] = manager
	p.ClusterConfig.Spec.Hosts = newHosts

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.AddPhases(
		&de.OverrideHostSudo{},
		&common.Connect{},
		&de.DetectOS{},
		&de.GatherFacts{},
		&de.ValidateHosts{},
		&de.DownloadBundle{},
		&common.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return fmt.Errorf("failed to download client bundle: %w", err)
	}
	return nil
}
