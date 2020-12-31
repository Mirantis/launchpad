package k0s

import (
	"os"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	k0s "github.com/Mirantis/mcc/pkg/product/k0s/phase"
	"gopkg.in/yaml.v2"
)

// Describe dumps information about the hosts
func (p *K0s) Describe(reportName string) error {

	if reportName == "config" {
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(p.ClusterConfig)
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&k0s.GatherFacts{},
		&common.Disconnect{},
		&k0s.Describe{},
	)
	return phaseManager.Run()
}
