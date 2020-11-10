package dockerenterprise

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	de "github.com/Mirantis/mcc/pkg/product/dockerenterprise/phase"
)

// ClientConfig downloads UCP client bundle
func (p *DockerEnterprise) ClientConfig() error {

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
