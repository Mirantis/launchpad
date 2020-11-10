package dockerenterprise

import (
	"os"

	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Describe - gets information about configured instance
func (p *DockerEnterprise) Describe(reportName string) error {
	var dtr bool
	var ucp bool

	if reportName == "dtr" {
		dtr = true
	}

	if reportName == "ucp" {
		ucp = true
	}

	log.Debugf("loaded cluster cfg: %+v", p.ClusterConfig)

	if reportName == "config" {
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(p.ClusterConfig)
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&phase.Connect{},
		&phase.GatherFacts{},
		&phase.Disconnect{},
		&phase.Describe{Ucp: ucp, Dtr: dtr},
	)

	return phaseManager.Run()
}
