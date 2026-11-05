package config

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/docker/hub"
	"github.com/Mirantis/launchpad/pkg/docker/reference"
	common "github.com/Mirantis/launchpad/pkg/product/common/config"
	validator "github.com/go-playground/validator/v10"
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
)

// ClusterMeta defines cluster metadata.
type ClusterMeta struct {
	Name string `yaml:"name" validate:"required"`
}

// ClusterConfig describes launchpad.yaml configuration.
//
// It marshals plainly: AdminPassword, host connection passwords, inline key
// material and the free-form flag lists are all ordinary marshalable values
// with no masking MarshalYAML, because the host credentials live on rig's
// types and cannot be given one here. Anything that encodes this struct for
// output must therefore pass the result through
// github.com/Mirantis/launchpad/pkg/util/redact. Today the only such path is
// MKE.Describe.
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
	validator.RegisterStructValidation(roleChecks, ClusterSpec{})
	if err := validator.RegisterValidation("image_repo", isValidImageRepo); err != nil {
		return fmt.Errorf("register image repo validation: %w", err)
	}
	if err := validator.Struct(c); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}
	return nil
}

// isValidImageRepo accepts only a value that composes into a well-formed
// Docker/OCI image reference.
//
// spec.*.imageRepo is expanded by envsubst and is interpolated into image
// references that become arguments to docker pull, tag, inspect and login on
// every host, so it is validated as an allowlist: anything the reference
// grammar does not describe is rejected rather than anything known-bad being
// denied. A denylist only blocks the metacharacters someone thought of.
//
// The grammar is pkg/docker/reference, not the regex at pkg/docker/image.go.
// That one is line-anchored and requires a tag and a slash because it scrapes
// bootstrapper stdout; it rejects "docker.io/mirantis", which launchpad ships
// as its own default.
func isValidImageRepo(fl validator.FieldLevel) bool {
	return reference.IsValidRepositoryPrefix(fl.Field().String())
}

func roleChecks(sl validator.StructLevel) {
	spec, ok := sl.Current().Interface().(ClusterSpec)
	if !ok {
		return
	}
	hosts := spec.Hosts
	if hosts.Count(func(h *Host) bool { return h.Role == "manager" }) == 0 {
		sl.ReportError(hosts, "hosts", "", "manager required", "")
	}
}

// Init returns an example of configuration file contents.
func Init(kind string) *ClusterConfig {
	mkeV, err := hub.LatestTag("mirantis", "ucp", false)
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
				Channel: constant.MCRChannel,
			},
			MKE: MKEConfig{
				Version: mkeV,
			},
			Hosts: []*Host{
				{
					Role: "manager",
					CompositeConfig: rig.CompositeConfig{
						SSH: &ssh.Config{
							Address: "10.0.0.1",
							User:    "root",
							Port:    22,
						},
					},
				},
				{
					Role: "worker",
					CompositeConfig: rig.CompositeConfig{
						SSH: &ssh.Config{
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
		msrV, err := hub.LatestTag("mirantis", "dtr", false)
		if err != nil {
			msrV = "required"
		}
		config.Spec.MSR = &MSRConfig{
			Version:    msrV,
			ReplicaIDs: "sequential",
		}

		config.Spec.Hosts = append(config.Spec.Hosts,
			&Host{
				Role: "msr",
				CompositeConfig: rig.CompositeConfig{
					SSH: &ssh.Config{
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
