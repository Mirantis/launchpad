package api

import (
	"github.com/Mirantis/mcc/pkg/product/common/api"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	validator "github.com/go-playground/validator/v10"
)

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/k0s/v1beta1"`
	Kind       string       `yaml:"kind" validate:"eq=k0s"`
	Metadata   *ClusterMeta `yaml:"metadata"`
	Spec       *ClusterSpec `yaml:"spec"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMeta{
		Name: "launchpad-k0s",
	}
	c.Spec = &ClusterSpec{}

	type clusterConfig ClusterConfig
	yc := (*clusterConfig)(c)

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
		APIVersion: "launchpad.mirantis.com/k0s/v1beta1",
		Kind:       kind,
		Metadata: &ClusterMeta{
			Name: "my-k0s-cluster",
		},
		Spec: &ClusterSpec{
			K0s: K0sConfig{
				Metadata: K0sMetadata{},
				Config:   api.GenericHash{}},
			Hosts: []*Host{
				{
					ConnectableHost: common.ConnectableHost{
						Address: "10.0.0.1",
						SSH: &common.SSH{
							User:    "root",
							Port:    22,
							KeyPath: "~/.ssh/id_rsa",
						},
					},
					Role: "manager",
				},
				{
					ConnectableHost: common.ConnectableHost{
						Address: "10.0.0.2",
						SSH:     common.DefaultSSH(),
					},
					Role: "worker",
				},
			},
		},
	}

	return config
}
