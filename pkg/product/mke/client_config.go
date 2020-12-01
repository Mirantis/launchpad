package mke

import (
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	de "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// ClientConfig downloads MKE client bundle
func (p *MKE) ClientConfig() error {

	manager := p.ClusterConfig.Spec.Managers()[0]
	newHosts := make(api.Hosts, 1)
	newHosts[0] = manager
	p.ClusterConfig.Spec.Hosts = newHosts

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.AddPhases(
		&common.Connect{},
		&de.GatherFacts{},
		&de.ValidateHosts{},
		&de.DownloadBundle{},
		&common.Disconnect{},
	)

	return phaseManager.Run()
}
