package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/urfave/cli/v2"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
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

	if err = addFileLogger(clusterConfig.Name); err != nil {
		return err
	}

	if err = clusterConfig.Validate(); err != nil {
		return err
	}

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
	phaseManager.AddPhase(&phase.JoinControllers{})
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

const fileMode = 0700

// adds a file logger too based on the cluster name
// The log path will be ~/.mirantis-mcc/<cluster-name>/install.log
// If cluster name is not given, the path will be ~/.mirantis-mcc/install.log
func addFileLogger(clusterName string) error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	baseDir := path.Join(home, ".mirantis-mcc", clusterName)
	if err = os.MkdirAll(baseDir, fileMode); err != nil {
		return err
	}
	logFileName := path.Join(baseDir, "install.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)

	if err != nil {
		return fmt.Errorf("Failed to create install log for cluster %s: %s", clusterName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed
	log.AddHook(&writer.Hook{
		Writer:    logFile,
		LogLevels: log.AllLevels,
	})

	return nil
}
