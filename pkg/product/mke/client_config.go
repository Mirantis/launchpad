package mke

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	de "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// ClientConfig downloads MKE client bundle.
func (p *MKE) ClientConfig() error {
	if p.ClusterConfig.Spec.MKE == nil {
		return fmt.Errorf("%w; Cannot download client bundle as there is no MKE installation", api.ErrThereIsNoMKE)
	}

	manager := p.ClusterConfig.Spec.Managers()[0]
	newHosts := make(api.Hosts, 1)
	newHosts[0] = manager
	p.ClusterConfig.Spec.Hosts = newHosts

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.AddPhases(
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
