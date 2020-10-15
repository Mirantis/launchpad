package describe

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// Describe shows data about cluster state
func Describe(configFile, reportName string) error {
	cfgData, err := config.ResolveClusterFile(configFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = config.Validate(&clusterConfig); err != nil {
		return err
	}

	var dtr bool
	var ucp bool

	if reportName == "dtr" {
		dtr = true
	}

	if reportName == "ucp" {
		ucp = true
	}

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{Dtr: dtr})
	phaseManager.AddPhase(&phase.Disconnect{})
	phaseManager.AddPhase(&phase.Describe{Ucp: ucp, Dtr: dtr})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}

	return nil
}
