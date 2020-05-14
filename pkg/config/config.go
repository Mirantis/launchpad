package config

import (
	"gopkg.in/yaml.v2"
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
	EngineRepoURL = "http://repos-internal.mirantis.com.s3.amazonaws.com"
	// EngineInstallURL is the default engine install script location
	EngineInstallURL = "https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh"
)

// ClusterConfig is the struct to read the cluster.yaml config into
type ClusterConfig struct {
	Hosts  Hosts        `yaml:"hosts"`
	Ucp    UcpConfig    `yaml:"ucp"`
	Engine EngineConfig `yaml:"engine"`

	ManagerJoinToken string
	WorkerJoinToken  string
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

// Controllers filters only the controllers from the cluster config
func (c *ClusterConfig) Controllers() []*Host {
	controllers := make([]*Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "controller" {
			controllers = append(controllers, h)
		}
	}

	return controllers
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig ClusterConfig
	raw := rawConfig{
		Engine: NewEngineConfig(),
		Ucp:    NewUcpConfig(),
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = ClusterConfig(raw)
	return nil
}
