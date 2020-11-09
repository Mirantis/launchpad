package dockerenterprise

import (
	"os"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	de "github.com/Mirantis/mcc/pkg/product/dockerenterprise/phase"
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
		&common.Connect{},
		&de.GatherFacts{},
		&common.Disconnect{},
		&de.Describe{Ucp: ucp, Dtr: dtr},
	)

	return phaseManager.Run()
}
