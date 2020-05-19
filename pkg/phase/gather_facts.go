package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"
	"github.com/cobaugh/osrelease"
	log "github.com/sirupsen/logrus"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts
type GatherFacts struct{}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run(conf *config.ClusterConfig) error {
	err := runParallelOnHosts(conf.Hosts, conf, investigateHost)
	if err != nil {
		return err
	}
	// Gather UCP related facts
	conf.Ucp.Metadata = &config.UcpMetadata{
		ClusterID:        "",
		Installed:        false,
		InstalledVersion: "",
	}
	swarmLeader := conf.Managers()[0]
	// If engine is installed, we can collect some UCP & Swarm related info too
	if swarmLeader.Metadata.EngineVersion != "" {
		ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
		}
		conf.Ucp.Metadata = ucpMeta
		if ucpMeta.Installed {
			log.Infof("%s: UCP has version %s", swarmLeader.Address, ucpMeta.InstalledVersion)
		} else {
			log.Infof("%s: UCP is not installed", swarmLeader.Address)
		}
		conf.Ucp.Metadata.ClusterID = swarm.ClusterID(swarmLeader)
	}

	err = p.validateFacts(conf)
	if err != nil {
		return err
	}
	return nil
}

// Validates the facts check out:
// - if swarm is already initialized its cluster ID matches with the one in local state
func (p *GatherFacts) validateFacts(config *config.ClusterConfig) error {
	if config.Ucp.Metadata != nil && config.Ucp.Metadata.ClusterID != config.State.ClusterID {
		return fmt.Errorf("cluster ID mismatch between local state (%s) and cluster state (%s). This configuration is probably for another cluster.", config.State.ClusterID, config.Ucp.Metadata.ClusterID)
	}

	log.Infof("Facts check out against local state, safe to confinue")
	return nil
}

func investigateHost(h *config.Host, c *config.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h.Address)

	os := &config.OsRelease{}
	if isWindows(h) {
		winOs, err := resolveWindowsOsRelease(h)
		if err != nil {
			return err
		}
		os = winOs
	} else {
		linuxOs, err := resolveLinuxOsRelease(h)
		if err != nil {
			return err
		}
		os = linuxOs
	}

	h.Metadata = &config.HostMetadata{
		Os: os,
	}
	err := resolveHostConfigurer(h)
	if err != nil {
		return err
	}
	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.InternalAddress = h.Configurer.ResolveInternalIP()
	h.Metadata.EngineVersion = resolveEngineVersion(h)

	log.Debugf("%s: internal address: %s", h.Address, h.Metadata.InternalAddress)

	return nil
}

func isWindows(h *config.Host) bool {
	// need to use STDIN so that we don't request PTY (which does not work on Windows)
	err := h.ExecCmd(`powershell`, `Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"`, false)
	if err != nil {
		return false
	}
	return true
}

func resolveWindowsOsRelease(h *config.Host) (*config.OsRelease, error) {
	osName, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").ProductName"`)
	osName = strings.TrimSpace(osName)
	osMajor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMajorVersionNumber"`)
	osMajor = strings.TrimSpace(osMajor)
	osMinor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMinorVersionNumber"`)
	osMinor = strings.TrimSpace(osMinor)
	osBuild, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentBuild"`)
	osBuild = strings.TrimSpace(osBuild)

	version := fmt.Sprintf("%s.%s.%s", osMajor, osMinor, osBuild)
	osRelease := &config.OsRelease{
		ID:      fmt.Sprintf("windows-%s", version),
		Name:    osName,
		Version: version,
	}

	return osRelease, nil
}

func resolveLinuxOsRelease(h *config.Host) (*config.OsRelease, error) {
	output, err := h.ExecWithOutput("cat /etc/os-release")
	if err != nil {
		return nil, err
	}
	info, err := osrelease.ReadString(output)
	if err != nil {
		return nil, err
	}
	osRelease := &config.OsRelease{
		ID:      info["ID"],
		IDLike:  info["ID_LIKE"],
		Name:    info["PRETTY_NAME"],
		Version: info["VERSION_ID"],
	}
	if osRelease.IDLike == "" {
		osRelease.IDLike = osRelease.ID
	}

	return osRelease, nil
}

func resolveHostConfigurer(h *config.Host) error {
	configurers := config.GetHostConfigurers()
	for _, resolver := range configurers {
		configurer := resolver(h)
		if configurer != nil {
			h.Configurer = configurer
		}
	}
	if h.Configurer == nil {
		return fmt.Errorf("%s: has unsupported OS (%s)", h.Address, h.Metadata.Os.Name)
	}
	return nil
}

func resolveEngineVersion(h *config.Host) string {
	cmd := h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`)
	version, err := h.ExecWithOutput(cmd)
	if err != nil {
		return ""
	}
	return version
}
