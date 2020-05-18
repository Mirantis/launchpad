package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/state"
	validator "github.com/go-playground/validator/v10"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	// ImageRepo is the default image repo to use
	ImageRepo = "docker.io/docker"
	// Version is the default UCP version to use
	Version = "3.3.0-rc1"
	// EngineVersion is the default engine version
	EngineVersion = "19.03.8-rc1"
	// EngineChannel is the default engine channel
	EngineChannel = "test"
	// EngineRepoURL is the default engine repo
	EngineRepoURL = "https://repos.mirantis.com"
	// EngineInstallURL is the default engine install script location
	EngineInstallURL = "https://get.mirantis.com/"
)

// ClusterConfig is the struct to read the cluster.yaml config into
type ClusterConfig struct {
	Hosts  Hosts        `yaml:"hosts" validate:"required,dive,min=1"`
	Ucp    UcpConfig    `yaml:"ucp"`
	Engine EngineConfig `yaml:"engine"`
	Name   string       `yaml:"name" validate:"required,min=3"`

	ManagerJoinToken string
	WorkerJoinToken  string
	State            *state.State
}

// FromYaml loads the cluster config from given yaml data
func FromYaml(data []byte) (ClusterConfig, error) {
	c := ClusterConfig{}

	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func (c *ClusterConfig) Validate() error {
	validator := validator.New()
	return validator.Struct(c)
}

// Workers filters only the workers from the cluster config
func (c *ClusterConfig) Workers() []*Host {
	workers := make([]*Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "worker" {
			workers = append(workers, h)
		}
	}

	return workers
}

// Managers filters only the manager nodes from the cluster config
func (c *ClusterConfig) Managers() []*Host {
	managers := make([]*Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "manager" {
			managers = append(managers, h)
		}
	}

	return managers
}

// ResolveClusterFile looks for the cluster.yaml file, based on the value, passed in ctx.
// It returns the contents of this file as []byte if found,
// or error if it didn't.
func ResolveClusterFile(ctx *cli.Context) ([]byte, error) {
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

// Helper for reading data from references to external files
var loadExternalFile = func(path string) ([]byte, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return []byte{}, err
	}

	filedata, err := ioutil.ReadFile(realpath)
	if err != nil {
		return []byte{}, err
	}
	return filedata, nil
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig ClusterConfig
	raw := rawConfig{
		Engine: NewEngineConfig(),
		Ucp:    NewUcpConfig(),
		Name:   "mcc-ucp",
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = ClusterConfig(raw)
	return nil
}
