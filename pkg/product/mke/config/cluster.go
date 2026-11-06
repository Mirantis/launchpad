package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/docker/hub"
	common "github.com/Mirantis/launchpad/pkg/product/common/config"
	validator "github.com/go-playground/validator/v10"
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
)

// ClusterMeta defines cluster metadata.
type ClusterMeta struct {
	// Name is joined verbatim into local filesystem paths -- the log
	// directory (cmd/common.go) and the client bundle directory
	// (pkg/mke/mke.go) -- so it is validated as a single path element.
	// A name carrying a separator or a dot-name is rejected rather than
	// rewritten: silently sanitising it would move a user's bundle and logs
	// somewhere they did not ask for, and could collapse two distinct
	// cluster names onto one directory.
	Name string `yaml:"name" validate:"required,cluster_name"`
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
	// Defence in depth, not the control: the command sites quote. Validation
	// makes a malformed value visible at config load instead of as a failed
	// install on every host.
	if err := validator.RegisterValidation("mcr_channel", isValidMCRChannel); err != nil {
		return fmt.Errorf("register mcr channel validation: %w", err)
	}
	if err := validator.RegisterValidation("mcr_repo_url", isValidMCRRepoURL); err != nil {
		return fmt.Errorf("register mcr repo url validation: %w", err)
	}
	if err := validator.RegisterValidation("cluster_name", isPathSafeName); err != nil {
		return fmt.Errorf("register cluster name validation: %w", err)
	}
	if err := validator.Struct(c); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}
	return nil
}

func isValidMCRChannel(fl validator.FieldLevel) bool {
	return common.IsValidMCRChannel(fl.Field().String())
}

func isValidMCRRepoURL(fl validator.FieldLevel) bool {
	return common.IsValidMCRRepoURL(fl.Field().String())
}

// isPathSafeName accepts only values usable as a single filesystem path
// element. The cluster name is joined into local paths, so a separator in
// either form, a dot-name, or a value that is only whitespace would place
// state outside the directory the user expects. "/" and "\" are both
// rejected regardless of host OS, so a config written on Linux cannot
// traverse when applied from Windows.
func isPathSafeName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if name == "" || name != strings.TrimSpace(name) {
		return false
	}
	if name == "." || name == ".." {
		return false
	}

	return !strings.ContainsAny(name, `/\`) && !strings.ContainsRune(name, os.PathSeparator)
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
