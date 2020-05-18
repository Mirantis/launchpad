package state

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// State defines the locally store cluster state
type State struct {
	ClusterID string `yaml:"clusterId"`
	Name      string `yaml:"name"`
}

// InitState ...
func InitState(clusterName string) (*State, error) {
	state := &State{
		Name: clusterName,
	}

	return state, state.Save()
}

// LoadState ...
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

// Save ...
func (s *State) Save() error {
	statePath, err := s.getStatePath()
	if err != nil {
		return err
	}

	if err = ensureDir(filepath.Dir(statePath)); err != nil {
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

func (s *State) getStatePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(home, constant.StateBaseDir, "cluster", s.Name, "state.yaml"), nil
}

// FIXME Needs to be not copy-pasta, I'll figure this out later...
func ensureDir(dirPath string) error {
	if _, serr := os.Stat(dirPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(dirPath, os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	return nil
}
