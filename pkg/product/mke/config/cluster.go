package config

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/docker/hub"
	common "github.com/Mirantis/launchpad/pkg/product/common/config"
	validator "github.com/go-playground/validator/v10"
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	log "github.com/sirupsen/logrus"
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
	c.warnInsecureWinRM()

	validator := validator.New(validator.WithRequiredStructEnabled())
	validator.RegisterStructValidation(roleChecks, ClusterSpec{})
	if err := validator.Struct(c); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}
	return nil
}

// warnInsecureWinRM warns about every host that combines winRM.insecure with a
// password written into the configuration file. winRM.insecure makes rig accept
// any server certificate, so the channel is encrypted but the peer is never
// authenticated: an on-path attacker can terminate the TLS session, recover the
// password and read every command issued to that host. This is deliberately a
// warning and not an error, because an operator may have accepted that risk
// knowingly and must still be able to apply. The message names the host and the
// settings only -- never the credential, nor its length or any digest of it.
func (c *ClusterConfig) warnInsecureWinRM() {
	if c.Spec == nil {
		return
	}

	for _, h := range c.Spec.Hosts {
		if h == nil || h.WinRM == nil {
			continue
		}
		wrm := h.WinRM
		if !wrm.UseHTTPS || !wrm.Insecure || wrm.Password == "" {
			continue
		}

		log.Warnf(
			"%s: winRM.insecure is true, so the host's TLS certificate is not verified, and winRM.password is set in the configuration file. "+
				"An on-path attacker can impersonate this host, capture the password and read every command sent to it. "+
				"Remove winRM.insecure once the host presents a trusted certificate, and keep the password out of the file by using an environment variable "+
				"(see docs/usage/getting-started.md, \"Keeping credentials out of launchpad.yaml\").",
			wrm.Address,
		)
	}
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
