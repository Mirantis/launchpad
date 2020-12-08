package api

import (
	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"empty=false"`
}

// ClusterConfig describes launchpad.yaml configuration
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"one_of=launchpad.mirantis.com/mke/v1.1"`
	Kind       string       `yaml:"kind" validate:"one_of=mke,mke+msr"`
	Metadata   *ClusterMeta `yaml:"metadata"`
	Spec       *ClusterSpec `yaml:"spec" validate:"nil=false"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMeta{
		Name: "launchpad-mke",
	}
	c.Spec = &ClusterSpec{}

	type spec ClusterConfig
	yc := (*spec)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}

// Init returns an example of configuration file contents
func Init(kind string) *ClusterConfig {
	config := &ClusterConfig{
		APIVersion: "launchpad.mirantis.com/mke/v1.1",
		Kind:       kind,
		Metadata: &ClusterMeta{
			Name: "my-mke-cluster",
		},
		Spec: &ClusterSpec{
			Engine: EngineConfig{
				Version: constant.EngineVersion,
			},
			MKE: MKEConfig{
				Version: constant.MKEVersion,
			},
			Hosts: []*Host{
				{
					Address: "10.0.0.1",
					Role:    "manager",
					SSH: &common.SSH{
						User:    "root",
						Port:    22,
						KeyPath: "~/.ssh/id_rsa",
					},
				},
				{
					Address: "10.0.0.2",
					Role:    "worker",
					SSH:     common.DefaultSSH(),
				},
			},
		},
	}
	if kind == "mke+msr" {
		config.Spec.MSR = &MSRConfig{
			Version:    constant.MSRVersion,
			ReplicaIDs: "sequential",
		}

		config.Spec.Hosts = append(config.Spec.Hosts,
			&Host{
				Address: "10.0.0.3",
				Role:    "msr",
				SSH:     common.DefaultSSH(),
			},
		)
	}

	return config
}
