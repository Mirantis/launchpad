package api

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/docker/hub"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	validator "github.com/go-playground/validator/v10"
	"github.com/k0sproject/rig"
)

// ClusterMeta defines cluster metadata.
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration.
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/mke/v1.6"`
	Kind       string       `yaml:"kind" validate:"oneof=mke mke+msr"`
	Metadata   *ClusterMeta `yaml:"metadata"`
	Spec       *ClusterSpec `yaml:"spec"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *ClusterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMeta{
		Name: "launchpad-mke",
	}
	c.Spec = &ClusterSpec{}

	type spec ClusterConfig
	yc := (*spec)(c)

	if err := unmarshal(yc); err != nil {
		return fmt.Errorf("failed to unmarshal cluster config: %w", err)
	}

	return nil
}

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func (c *ClusterConfig) Validate() error {
	validator := validator.New(validator.WithRequiredStructEnabled())
	validator.RegisterStructValidation(validateClusterSpec, ClusterSpec{})
	if err := validator.Struct(c); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}

	return nil
}

func validateClusterSpec(vsl validator.StructLevel) {
	spec, ok := vsl.Current().Interface().(ClusterSpec)

	if !ok {
		vsl.ReportError(nil, "spec", "", "no spec in cluster", "")
		return
	}

	if len(spec.Managers()) == 0 {
		vsl.ReportError(spec.Hosts, "hosts", "", "at least one manager host required", "")
	}

	if spec.MSR2 != nil && len(spec.MSR2s()) == 0 {
		vsl.ReportError(spec.Hosts, "hosts", "", "with msr2 configuration, at least one msr2 host is required", "")
	}

	if spec.MKE == nil && (spec.MSR2 != nil || spec.MSR3 != nil) {
		vsl.ReportError(spec, "spec", "", "if msr2 or msr3 installation is requested, mke must also be included in the installation instructions", "")
	}
}

// Init returns an example of configuration file contents.
func Init(kind string) *ClusterConfig {
	mkeV, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "ucp", false)
	if err != nil {
		mkeV = "required"
	}

	config := &ClusterConfig{
		APIVersion: "launchpad.mirantis.com/mke/v1.6",
		Kind:       kind,
		Metadata: &ClusterMeta{
			Name: "my-mke-cluster",
		},
		Spec: &ClusterSpec{
			MCR: common.MCRConfig{
				Version: constant.MCRVersion,
			},
			MKE: &MKEConfig{
				Version: mkeV,
			},
			Hosts: []*Host{
				{
					Role: "manager",
					Connection: rig.Connection{
						SSH: &rig.SSH{
							Address: "10.0.0.1",
							User:    "root",
							Port:    22,
						},
					},
				},
				{
					Role: "worker",
					Connection: rig.Connection{
						SSH: &rig.SSH{
							Address: "10.0.0.2",
							User:    "root",
							Port:    22,
						},
					},
				},
			},
		},
	}
	if kind == "mke+msr" {
		msrV, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "dtr", false)
		if err != nil {
			msrV = "required"
		}
		config.Spec.MSR2 = &MSR2Config{
			Version:    msrV,
			ReplicaIDs: "sequential",
		}

		config.Spec.Hosts = append(config.Spec.Hosts,
			&Host{
				Role: "msr",
				Connection: rig.Connection{
					SSH: &rig.SSH{
						Address: "10.0.0.2",
						User:    "root",
						Port:    22,
					},
				},
			},
		)
	}

	return config
}
