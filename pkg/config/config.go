package config

import (
	"gopkg.in/yaml.v2"
)

const (
	ImageRepo = "docker.io/docker"
	Version   = "3.3.0-rc.1"

	EngineVersion    = "19.03.8-rc.1"
	EngineChannel    = "test"
	EngineRepoURL    = "http://repos-internal.mirantis.com.s3.amazonaws.com"
	EngineInstallURL = "https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh"
)

type ClusterConfig struct {
	Hosts  Hosts        `yaml:"hosts"`
	Ucp    UcpConfig    `yaml:"ucp"`
	Engine EngineConfig `yaml:"engine"`

	ManagerJoinToken string
	WorkerJoinToken  string
}

func FromYaml(data []byte) (ClusterConfig, error) {
	c := ClusterConfig{}

	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}

func (c *ClusterConfig) Workers() []*Host {
	workers := make([]*Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "worker" {
			workers = append(workers, h)
		}
	}

	return workers
}

func (c *ClusterConfig) Controllers() []*Host {
	controllers := make([]*Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "controller" {
			controllers = append(controllers, h)
		}
	}

	return controllers
}

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
