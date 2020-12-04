package dummy

import (
	"os"

	"github.com/Mirantis/mcc/pkg/phase"
	dum "github.com/Mirantis/mcc/pkg/product/dummy/phase"
	"gopkg.in/yaml.v2"
)

// Describe - gets information about configured instance
func (p *Dummy) Describe(reportName string) error {
	// This needs to be in the parent cmd
	if reportName == "config" {
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(p.ClusterConfig)
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&dum.Connect{},
		&dum.Describe{},
		&dum.Disconnect{},
	)

	return phaseManager.Run()
}
