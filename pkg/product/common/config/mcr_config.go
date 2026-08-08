package config

import (
	"github.com/Mirantis/launchpad/pkg/constant"
)

type DockerInfo struct {
	ServerVersion string `json:"ServerVersion"`
	APIVersion    string `json:"APIVersion"`
	OS            string `json:"OperatingSystem"`
	KernelVersion string `json:"KernelVersion"`
	DockerRootDir string `json:"DockerRootDir"`
}

type DockerDaemonConfig struct {
	ExecRoot string `json:"exec-root"`
	Root     string `json:"root-data"`
}

// MCRConfig holds the Mirantis Container Runtime installation specific options.
type MCRConfig struct {
	RepoURL                     string   `yaml:"repoURL,omitempty"`
	AdditionalRuntimes          string   `yaml:"additionalRuntimes,omitempty"`
	DefaultRuntime              string   `yaml:"defaultRuntime,omitempty"`
	License                     string   `yaml:"license"`
	InstallScriptRemoteDirLinux string   `yaml:"installScriptRemoteDirLinux,omitempty"`
	InstallURLWindows           string   `yaml:"installURLWindows,omitempty"`
	Channel                     string   `yaml:"channel,omitempty"`
	Prune                       bool     `yaml:"prune,omitempty"`
	ForceUpgrade                bool     `yaml:"forceUpgrade,omitempty"`
	SwarmInstallFlags           Flags    `yaml:"swarmInstallFlags,omitempty,flow"`
	SwarmUpdateCommands         []string `yaml:"swarmUpdateCommands,omitempty,flow"`

	// InstallCLI additionally installs the docker CLI package (docker-ee-cli)
	// alongside the runtime, in the same package manager transaction so the two
	// cannot end up at mismatched versions.
	//
	// The runtime package lists the CLI only as a Recommends (a weak
	// dependency), not a Requires/Depends, on every repos.mirantis.com package
	// checked -- rpm (dnf/yum) and deb (apt) alike. Package managers install
	// Recommends by default, so this is normally not needed. It is needed when
	// weak-dependency installation is disabled, which common hardening
	// baselines do explicitly (dnf/yum: install_weak_deps=false; apt:
	// APT::Install-Recommends "false"). Set it on hosts where that applies, or
	// where the runtime has otherwise installed without the CLI. See
	// PRODENG-3641.
	//
	// Linux only. Windows installs MCR through install.ps1, which ships the CLI
	// as part of the same archive and has no separate package, so this value is
	// ignored on Windows hosts.
	InstallCLI bool `yaml:"installCLI,omitempty"`

	Metadata *MCRMetadata `yaml:"-"`
}

type MCRMetadata struct {
	ManagerJoinToken string
	WorkerJoinToken  string
	MCRChannel       string
}

// UnmarshalYAML puts in sane defaults when unmarshaling from yaml.
func (c *MCRConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type mcrConfig MCRConfig
	c.Metadata = &MCRMetadata{}
	yc := (*mcrConfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	c.SetDefaults()

	return nil
}

// SetDefaults sets defaults on the object.
func (c *MCRConfig) SetDefaults() {
	// Constants can't be used in tags, so yaml defaults can't be used here.
	// Note: Channel intentionally has no default — it is required and must be
	// set explicitly. See ClusterSpec.UnmarshalYAML for the validation.

	if c.RepoURL == "" {
		c.RepoURL = constant.MCRRepoURL
	}

	if c.InstallURLWindows == "" {
		c.InstallURLWindows = constant.MCRInstallURLWindows
	}
}
