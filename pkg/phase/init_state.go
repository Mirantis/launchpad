package phase

import (
	"fmt"
	"os"
	"path"

	"github.com/Mirantis/mcc/pkg/config"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/state"
	log "github.com/sirupsen/logrus"
)

// InitState loads or initializes the state
type InitState struct{}

// Title title for the phase
func (p *InitState) Title() string {
	return "Load or init local state"
}

// Run runs the state management logic
func (p *InitState) Run(config *config.ClusterConfig) error {
	localState, err := state.LoadState(config.Name)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Local state not found, initializing")
			localState, err = state.InitState(config.Name)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	config.State = localState
	log.Debugf("Initialized local state")
	stateDir, err := localState.GetDir()
	if err != nil {
		return err
	}
	log.Infof("Starting to send debug logs to: %s", stateDir+"/install.log")
	err = addFileLogger(stateDir)
	if err != nil {
		return err
	}
	return nil
}

const fileMode = 0700

// adds a file logger too based on the cluster name
// The log path will be ~/.mirantis-mcc/<cluster-name>/install.log
// If cluster name is not given, the path will be ~/.mirantis-mcc/install.log
func addFileLogger(stateDir string) error {

	logFileName := path.Join(stateDir, "install.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)

	if err != nil {
		return fmt.Errorf("Failed to create install log at %s: %s", logFileName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed
	log.AddHook(mcclog.NewFileHook(logFile))

	return nil
}
