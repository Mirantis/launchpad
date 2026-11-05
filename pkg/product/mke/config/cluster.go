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
	c.warnWinRMPasswordExposure()

	validator := validator.New(validator.WithRequiredStructEnabled())
	validator.RegisterStructValidation(roleChecks, ClusterSpec{})
	if err := validator.Struct(c); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}
	return nil
}

// warnWinRMPasswordExposure warns about every host whose WinRM password is
// exposed by the transport it is configured to use. Two distinct failures are
// reported separately, because the remedy differs:
//
//   - plain HTTP with a password. rig installs a transport decorator only for
//     NTLM, or for HTTPS with a loaded client certificate
//     (protocol/winrm/connection.go:242-250). With neither, go-winrm's default
//     transport calls req.SetBasicAuth (masterzen/winrm/http.go:96), so the
//     password crosses the network base64-encoded in an Authorization header
//     with no encryption at all.
//   - HTTPS with winRM.insecure. The channel is encrypted but the peer is
//     never authenticated, so an on-path attacker can impersonate the host,
//     recover the password and read every command issued to it.
//
// Both are warnings rather than errors: an operator may have accepted either
// risk knowingly and must still be able to apply. The messages name the host
// and the settings only -- never the credential, nor its length or any digest.
func (c *ClusterConfig) warnWinRMPasswordExposure() {
	if c.Spec == nil {
		return
	}

	for _, h := range c.Spec.Hosts {
		if h == nil || h.WinRM == nil {
			continue
		}
		wrm := h.WinRM
		if wrm.Password == "" {
			continue
		}

		// No message-level protection either: NTLM negotiates its own
		// encryption, so it is not cleartext even over plain HTTP.
		if !wrm.UseHTTPS && !wrm.UseNTLM {
			log.Warnf(
				"%s: winRM.useHTTPS is false, so the WinRM password is sent in clear. "+
					"It is base64-encoded in an HTTP Authorization header, which is encoding, not encryption, "+
					"and anyone on the network path can read it along with every command sent to this host. "+
					"Set winRM.useHTTPS and give the host a certificate your client trusts "+
					"(see docs/usage/getting-started.md, \"WinRM over TLS\").",
				wrm.Address,
			)

			continue
		}

		// The message cannot tell whether the password was inlined or came
		// from an environment variable: substitution happens at
		// pkg/config/config.go:79, before the config is parsed, so the field
		// is populated identically either way. It therefore leads with the
		// certificate problem, which is always actionable, and mentions
		// sourcing the password from the environment as secondary advice
		// rather than implying the user has inlined it.
		if wrm.UseHTTPS && wrm.Insecure {
			log.Warnf(
				"%s: winRM.insecure is true, so the host's TLS certificate is not verified. "+
					"An on-path attacker can impersonate this host, capture the WinRM password and read every command sent to it. "+
					"Remove winRM.insecure once the host presents a trusted certificate (pin it with winRM.caCertPath), "+
					"and source the password from the environment rather than inlining it "+
					"(see docs/usage/getting-started.md, \"Keeping credentials out of launchpad.yaml\").",
				wrm.Address,
			)
		}
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
