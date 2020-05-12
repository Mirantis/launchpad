package config

import (
	"github.com/Mirantis/mcc/pkg/host"
	"gopkg.in/yaml.v2"
)

type ClusterConfig struct {
	Hosts host.Hosts `yaml:"hosts"`

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

func (c *ClusterConfig) Workers() []*host.Host {
	workers := make([]*host.Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "worker" {
			workers = append(workers, h)
		}
	}

	return workers
}

func (c *ClusterConfig) Controllers() []*host.Host {
	controllers := make([]*host.Host, 0)
	for _, h := range c.Hosts {
		if h.Role == "controller" {
			controllers = append(controllers, h)
		}
	}

	return controllers
}
