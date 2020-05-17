package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/urfave/cli/v2"

	log "github.com/sirupsen/logrus"
)

// Install ...
func Install(ctx *cli.Context) error {
	cfgData, err := resolveClusterFile(ctx)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = clusterConfig.Validate(); err != nil {
		return err
	}

	fmt.Println(util.Logo)

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)

	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherHostFacts{})
	phaseManager.AddPhase(&phase.PrepareHost{})
	phaseManager.AddPhase(&phase.InstallEngine{})
	phaseManager.AddPhase(&phase.GatherUcpFacts{})
	phaseManager.AddPhase(&phase.PullImages{})
	phaseManager.AddPhase(&phase.InitSwarm{})
	phaseManager.AddPhase(&phase.InstallUCP{})
	phaseManager.AddPhase(&phase.UpgradeUcp{})
	phaseManager.AddPhase(&phase.JoinManagers{})
	phaseManager.AddPhase(&phase.JoinWorkers{})
	phaseManager.AddPhase(&phase.Disconnect{})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}

	return nil

}

func resolveClusterFile(ctx *cli.Context) ([]byte, error) {
	clusterFile := ctx.String("config")
	fp, err := filepath.Abs(clusterFile)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to lookup current directory name: %v", err)
	}
	file, err := os.Open(fp)
	if err != nil {
		return []byte{}, fmt.Errorf("can not find cluster configuration file: %v", err)
	}
	log.Debugf("opened config file from %s", fp)

	defer file.Close()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file: %v", err)
	}
	return buf, nil
}
