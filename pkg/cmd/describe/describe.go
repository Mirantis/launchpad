package describe

import (
	"fmt"
	"os"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"

	log "github.com/sirupsen/logrus"
)

// Describe ...
func Describe(configFile string) error {
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

	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Stdout.WriteString(util.Logo)
		os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
	}

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)

	dtr := config.ContainsDtr(clusterConfig)

	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{Dtr: dtr})
	phaseManager.AddPhase(&phase.Disconnect{})
	phaseManager.AddPhase(&phase.Describe{})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}

	return nil
}
