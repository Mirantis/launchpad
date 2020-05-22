package state

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// State defines the locally store cluster state
type State struct {
	ClusterID string `yaml:"clusterId"`
	Name      string `yaml:"name"`
}

// InitState initilizes and saves the "empty" state
func InitState(clusterName string) (*State, error) {
	state := &State{
		Name: clusterName,
	}

	return state, state.Save()
}

// LoadState loads existing local state from disk
func LoadState(clusterName string) (*State, error) {
	state := &State{
		Name: clusterName,
	}

	statePath, err := state.getStatePath()
	if err != nil {
		return nil, err
	}
	// Make sure we can read the file, or return sensible NotExist error
	if _, err := os.Stat(statePath); err != nil {
		return state, err
	}
	log.Debugf("loading local state from %s", statePath)

	filedata, err := ioutil.ReadFile(statePath)
	err = yaml.Unmarshal(filedata, state)
	if err != nil {
		return state, err
	}
	return state, nil
}

// Save saves the state on disk
func (s *State) Save() error {
	statePath, err := s.getStatePath()
	if err != nil {
		return err
	}

	if err = util.EnsureDir(filepath.Dir(statePath)); err != nil {
		return err
	}
	d, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(statePath, d, 0644)
	if err != nil {
		return err
	}
	return nil
}

// GetDir returns the clusters state directory
func (s *State) GetDir() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(home, constant.StateBaseDir, "cluster", s.Name), nil
}

// gets the full path to the clusters state file
func (s *State) getStatePath() (string, error) {
	stateDir, err := s.GetDir()
	if err != nil {
		return "", err
	}
	return path.Join(stateDir, "state.yaml"), nil
}
