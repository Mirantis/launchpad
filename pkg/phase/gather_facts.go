package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"
	"github.com/cobaugh/osrelease"
	log "github.com/sirupsen/logrus"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts
type GatherFacts struct {
	Analytics
	BasicPhase
	Dtr bool
}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run() error {
	err := runParallelOnHosts(p.config.Spec.Hosts, p.config, p.investigateHost)
	if err != nil {
		return err
	}
	// Gather UCP related facts

	swarmLeader := p.config.Spec.SwarmLeader()

	// If engine is installed, we can collect some UCP & Swarm related info too
	if swarmLeader.Metadata.EngineVersion != "" {
		err := ucp.CollectUcpFacts(swarmLeader, p.config.Spec.Ucp.Metadata)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
		}
		if p.config.Spec.Ucp.Metadata.Installed {
			log.Infof("%s: UCP has version %s", swarmLeader.Address, p.config.Spec.Ucp.Metadata.InstalledVersion)
		} else {
			log.Infof("%s: UCP is not installed", swarmLeader.Address)
		}
		p.config.Spec.Ucp.Metadata.ClusterID = swarm.ClusterID(swarmLeader)
	}
	if p.Dtr {
		// If we intend to configure DTR as well, gather facts for DTR
		if p.config.Spec.Dtr == nil {
			p.config.Spec.Dtr = &api.DtrConfig{}
		}

		p.config.Spec.Dtr.Metadata = &api.DtrMetadata{
			Installed:          false,
			InstalledVersion:   "",
			DtrLeaderReplicaID: "",
		}
		dtrLeader := p.config.Spec.DtrLeader()
		if dtrLeader != nil && dtrLeader.Metadata != nil && dtrLeader.Metadata.EngineVersion != "" {
			dtrMeta, err := dtr.CollectDtrFacts(dtrLeader)
			if err != nil {
				return fmt.Errorf("%s: failed to collect existing DTR details: %s", dtrLeader.Address, err.Error())
			}
			p.config.Spec.Dtr.Metadata = dtrMeta
			if dtrMeta.Installed {
				log.Infof("%s: DTR has version %s", dtrLeader.Address, dtrMeta.InstalledVersion)
			} else {
				log.Infof("%s: DTR is not installed", dtrLeader.Address)
			}
		}
	}

	return nil
}

func (p *GatherFacts) investigateHost(h *api.Host, c *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h.Address)

	os := &api.OsRelease{}
	if p.isWindows(h) {
		h.Connection.SetWindows(true)
		winOs, err := p.resolveWindowsOsRelease(h)
		if err != nil {
			return err
		}
		os = winOs
	} else {
		linuxOs, err := p.resolveLinuxOsRelease(h)
		if err != nil {
			return err
		}
		os = linuxOs
	}

	h.Metadata = &api.HostMetadata{
		Os: os,
	}
	if err := api.ResolveHostConfigurer(h); err != nil {
		return err
	}

	if err := h.Configurer.CheckPrivilege(); err != nil {
		return err
	}

	h.Metadata.EngineVersion = h.EngineVersion()

	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.LongHostname = h.Configurer.ResolveLongHostname()
	a, err := h.Configurer.ResolveInternalIP()
	if err != nil {
		return err
	}
	h.Metadata.InternalAddress = a

	log.Infof("%s: is running \"%s\"", h.Address, h.Metadata.Os.Name)
	log.Infof("%s: internal address: %s", h.Address, h.Metadata.InternalAddress)

	log.Infof("%s: gathered all facts", h.Address)
	return nil
}

func (p *GatherFacts) isWindows(h *api.Host) bool {
	return h.ExecCmd("cmd /c exit 0", "", false, false) == nil
}

// ResolveWindowsOsRelease ...
func (p *GatherFacts) resolveWindowsOsRelease(h *api.Host) (*api.OsRelease, error) {
	osName, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").ProductName"`)
	osName = strings.TrimSpace(osName)
	osMajor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMajorVersionNumber"`)
	osMajor = strings.TrimSpace(osMajor)
	osMinor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMinorVersionNumber"`)
	osMinor = strings.TrimSpace(osMinor)
	osBuild, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentBuild"`)
	osBuild = strings.TrimSpace(osBuild)

	version := fmt.Sprintf("%s.%s.%s", osMajor, osMinor, osBuild)
	osRelease := &api.OsRelease{
		ID:      fmt.Sprintf("windows-%s", version),
		Name:    osName,
		Version: version,
	}

	return osRelease, nil
}

// ResolveLinuxOsRelease ...
func (p *GatherFacts) resolveLinuxOsRelease(h *api.Host) (*api.OsRelease, error) {
	output, err := h.ExecWithOutput("cat /etc/os-release")
	if err != nil {
		return nil, err
	}
	info, err := osrelease.ReadString(output)
	if err != nil {
		return nil, err
	}
	osRelease := &api.OsRelease{
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

func (p *GatherFacts) testConnection(h *api.Host) error {
	testfn := "launchpad_connection_test.txt"

	// cleanup
	if h.Configurer.FileExist(testfn) {
		if err := h.Configurer.DeleteFile(testfn); err != nil {
			return fmt.Errorf("failed to delete connection test file: %w", err)
		}
	}

	if err := h.Configurer.WriteFile(testfn, "test", "0600"); err != nil {
		return fmt.Errorf("failed to write connection test file: %w", err)
	}

	if !h.Configurer.FileExist(testfn) {
		return fmt.Errorf("file does not exist after connection test file write")
	}

	content, err := h.Configurer.ReadFile(testfn)
	if content != "test" || err != nil {
		h.Configurer.DeleteFile(testfn)

		return fmt.Errorf(`connection file write test failed, expected "test", received "%s" (%w)`, content, err)
	}

	err = h.Configurer.DeleteFile(testfn)
	if err != nil || h.Configurer.FileExist(testfn) {
		return fmt.Errorf("connection file write test failed at file exist after delete check")
	}

	return nil
}
