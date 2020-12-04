package api

import (
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	validator "github.com/go-playground/validator/v10"
)

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/dummy/v0.1-beta"`
	Kind       string       `yaml:"kind" validate:"eq=dummy"`
	Metadata   *ClusterMeta `yaml:"metadata"`
	Spec       *ClusterSpec `yaml:"spec"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMeta{
		Name: "launchpad-dummy",
	}
	c.Spec = &ClusterSpec{}

	type spec ClusterConfig
	yc := (*spec)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func (c *ClusterConfig) Validate() error {
	validator := validator.New()
	return validator.Struct(c)
}

// Init returns an example of configuration file contents
func Init(kind string) *ClusterConfig {
	config := &ClusterConfig{
		APIVersion: "launchpad.mirantis.com/dummy/v0.1-beta",
		Kind:       kind,
		Metadata: &ClusterMeta{
			Name: "my-dummy-cluster",
		},
		Spec: &ClusterSpec{
			Hosts: []*Host{
				{
					Address: "10.0.0.1",
					SSH: &common.SSH{
						User:    "root",
						Port:    22,
						KeyPath: "~/.ssh/id_rsa",
					},
				},
				{
					Address: "10.0.0.2",
					SSH:     common.DefaultSSH(),
				},
			},
		},
	}

	return config
}
