package api

import (
	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	validator "github.com/go-playground/validator/v10"
	"github.com/k0sproject/rig"
)

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/mke/v1.3"`
	Kind       string       `yaml:"kind" validate:"oneof=mke mke+msr"`
	Metadata   *ClusterMeta `yaml:"metadata"`
	Spec       *ClusterSpec `yaml:"spec"`
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

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func (c *ClusterConfig) Validate() error {
	validator := validator.New()
	validator.RegisterStructValidation(roleChecks, ClusterSpec{})
	return validator.Struct(c)
}

func roleChecks(sl validator.StructLevel) {
	spec := sl.Current().Interface().(ClusterSpec)
	hosts := spec.Hosts
	if hosts.Count(func(h *Host) bool { return h.Role == "manager" }) == 0 {
		sl.ReportError(hosts, "hosts", "", "manager required", "")
	}
}

// Init returns an example of configuration file contents
func Init(kind string) *ClusterConfig {
	config := &ClusterConfig{
		APIVersion: "launchpad.mirantis.com/mke/v1.3",
		Kind:       kind,
		Metadata: &ClusterMeta{
			Name: "my-mke-cluster",
		},
		Spec: &ClusterSpec{
			MCR: common.MCRConfig{
				Version: constant.MCRVersion,
			},
			MKE: MKEConfig{
				Version: constant.MKEVersion,
			},
			Hosts: []*Host{
				{
					Role: "manager",
					Connection: rig.Connection{
						SSH: &rig.SSH{
							Address: "10.0.0.1",
							User:    "root",
							Port:    22,
							KeyPath: "~/.ssh/id_rsa",
						},
					},
				},
				{
					Role: "worker",
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
				Role: "msr",
			},
		)
	}

	return config
}
