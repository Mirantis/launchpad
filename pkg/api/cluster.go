package api

import (
	"github.com/Mirantis/mcc/pkg/constant"
	validator "github.com/go-playground/validator/v10"
)

// ClusterMeta defines cluster metadata
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration
type ClusterConfig struct {
	APIVersion string       `yaml:"apiVersion" validate:"eq=launchpad.mirantis.com/mke/v1.1"`
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
	validator.RegisterStructValidation(requireManager, ClusterSpec{})
	validator.RegisterStructValidation(disallowMSR, ClusterConfig{})
	return validator.Struct(c)
}

func requireManager(sl validator.StructLevel) {
	hosts := sl.Current().Interface().(ClusterSpec).Hosts
	if hosts.Count(func(h *Host) bool { return h.Role == "manager" }) == 0 {
		sl.ReportError(hosts, "hosts", "", "manager required", "")
	}
}

// disallow MSR config when using the "mke" kind instead of "mke+msr"
func disallowMSR(sl validator.StructLevel) {
	if sl.Current().Interface().(ClusterConfig).Kind == "mke+msr" {
		return
	}

	spec := sl.Current().Interface().(ClusterConfig).Spec
	if spec.MSR != nil {
		sl.ReportError(spec.MSR, "msr", "", "msr configuration is only available with kind: mke+msr", "")
	}

	hosts := spec.Hosts
	if hosts.Count(func(h *Host) bool { return h.Role == "msr" }) > 0 {
		sl.ReportError(hosts, "hosts", "", "role=msr is only available with kind: mke+msr", "")
	}
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
					SSH: &SSH{
						User:    "root",
						Port:    22,
						KeyPath: "~/.ssh/id_rsa",
					},
				},
				{
					Address: "10.0.0.2",
					Role:    "worker",
					SSH:     DefaultSSH(),
				},
			},
		},
	}
	if kind == "mke+msr" {
		config.Spec.MSR = &MSRConfig{
			Version:       constant.MSRVersion,
			ReplicaConfig: "sequential",
		}

		config.Spec.Hosts = append(config.Spec.Hosts,
			&Host{
				Address: "10.0.0.3",
				Role:    "msr",
				SSH:     DefaultSSH(),
			},
		)
	}

	return config
}
