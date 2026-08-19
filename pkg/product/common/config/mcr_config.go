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

	// InstallRecommends tells the package manager to install the runtime's
	// recommended packages even when the host is configured not to.
	//
	// docker-ee declares docker-ee-cli and cri-dockerd-ee as recommended
	// packages (rpm Recommends / deb Recommends), not as hard requirements.
	// Package managers install recommended packages by default, so this is
	// normally unnecessary. Hardening baselines commonly turn that off --
	// install_weak_deps=false for dnf/yum, APT::Install-Recommends "false" for
	// apt, solver.onlyRequires for zypper -- and the runtime then installs
	// without its CLI, leaving later docker commands to fail on a host that
	// otherwise looks correctly installed. See PRODENG-3641.
	//
	// Note the set of recommended packages is defined by repository metadata,
	// not by launchpad, and can differ by platform and change over time. On rpm
	// hosts it typically includes docker-ee-cli and cri-dockerd-ee. On deb hosts
	// it may include additional packages (including a kernel package).
	//
	// Linux only. Windows installs MCR through install.ps1 from a single archive
	// that already contains the CLI; there is no package manager and no
	// equivalent concept, so this value is ignored on Windows hosts.
	InstallRecommends bool `yaml:"installRecommends,omitempty"`

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
