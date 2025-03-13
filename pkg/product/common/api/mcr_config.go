package api

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
	Version                     string   `yaml:"version"`
	RepoURL                     string   `yaml:"repoURL,omitempty"`
	AdditionalRuntimes          string   `yaml:"additionalRuntimes,omitempty"`
	DefaultRuntime              string   `yaml:"defaultRuntime,omitempty"`
	InstallURLLinux             string   `yaml:"installURLLinux,omitempty"`
	InstallScriptRemoteDirLinux string   `yaml:"installScriptRemoteDirLinux,omitempty"`
	InstallURLWindows           string   `yaml:"installURLWindows,omitempty"`
	Channel                     string   `yaml:"channel,omitempty"`
	Prune                       bool     `yaml:"prune,omitempty"`
	ForceUpgrade                bool     `yaml:"forceUpgrade,omitempty"`
	SwarmInstallFlags           Flags    `yaml:"swarmInstallFlags,omitempty,flow"`
	SwarmUpdateCommands         []string `yaml:"swarmUpdateCommands,omitempty,flow"`

	Metadata *MCRMetadata `yaml:"-"`
}

type MCRMetadata struct {
	ManagerJoinToken string
	WorkerJoinToken  string
}

// UnmarshalYAML puts in sane defaults when unmarshaling from yaml.
func (c *MCRConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
	if c.Version == "" {
		c.Version = constant.MCRVersion
	}

	if c.Channel == "" {
		c.Channel = constant.MCRChannel
	}

	if c.RepoURL == "" {
		c.RepoURL = constant.MCRRepoURL
	}

	if c.InstallURLLinux == "" {
		c.InstallURLLinux = constant.MCRInstallURLLinux
	}

	if c.InstallURLWindows == "" {
		c.InstallURLWindows = constant.MCRInstallURLWindows
	}
}
