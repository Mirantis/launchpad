package dockerenterprise

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
)

// ClientConfig downloads UCP client bundle
func (p *DockerEnterprise) ClientConfig() error {

	manager := p.ClusterConfig.Spec.Managers()[0]
	newHosts := make(api.Hosts, 1)
	newHosts[0] = manager
	p.ClusterConfig.Spec.Hosts = newHosts

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.AddPhases(
		&phase.Connect{},
		&phase.GatherFacts{},
		&phase.ValidateHosts{},
		&phase.DownloadBundle{},
		&phase.Disconnect{},
	)

	return phaseManager.Run()
}
