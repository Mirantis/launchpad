package api

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Mirantis/launchpad/pkg/constant"
	version "github.com/hashicorp/go-version"
)

var (
	ErrInvalidVersion = errors.New("the MCR version is not valid")
	// all versions from 25.0.0 need channel-version matching.
	minVersionNeedsMatchingChannel, _ = version.NewVersion("25.0.0")
	ErrChannelDoesntMatchVersion      = errors.New("MCR version and channel don't match, which is required for versions >= 25.0.0")

	fipsChannelSuffix = "/fips"                            // this suffix is removed from channels for version comparison testing
	suffixPattern     = regexp.MustCompile(`-(tp|rc)\d+$`) // this filters out internal build suffix like -tp1
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
	License                     string   `yaml:"license"`
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

// Validate mcr config values.
func (c *MCRConfig) Validate() error {
	if err := processVersionChannelMatch(c); err != nil {
		return err
	}

	return nil
}

// MCR versions 25.0 and later require that the channel uses the version specific part.
//
//	If the channel doesn't contain the right version component then version pinning won't work
func processVersionChannelMatch(config *MCRConfig) error {
	ver, vererr := processVersionIsAVersion(config)
	if vererr != nil {
		return fmt.Errorf("%w; %w", ErrInvalidVersion, vererr)
	}

	if ver.LessThan(minVersionNeedsMatchingChannel) {
		return nil
	}

	channel := strings.TrimSuffix(config.Channel, fipsChannelSuffix)
	chanParts := strings.Split(channel, "-")
	if len(chanParts) == 1 {
		return fmt.Errorf("%w; channel has no version (%s)", ErrChannelDoesntMatchVersion, config.Channel)
	}

	if len(chanParts) > 2 {
		return fmt.Errorf("%w; channel parts could not be interpreted", ErrChannelDoesntMatchVersion)
	}

	configVerNum := suffixPattern.ReplaceAllString(config.Version, "") // remove build number
	if !strings.HasPrefix(chanParts[1], configVerNum) {
		return fmt.Errorf("%w; MCR version does not match channel-version '%s' vs '%s'", ErrChannelDoesntMatchVersion, chanParts[1], config.Version)
	}

	return nil
}

// go-version.NewVersion throws a runtime error if you pass it something invalid
// so we use, this to provide a runtime safe process for the version parsing.
func processVersionIsAVersion(config *MCRConfig) (ver *version.Version, err error) {
	if config.Version == "" {
		err = ErrInvalidVersion
		return ver, err
	}

	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if recover() != nil {
			err = ErrInvalidVersion
		}
	}()

	ver, err = version.NewVersion(config.Version)
	return ver, err //nolint:wrapcheck
}
