package phase

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
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
}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run(conf *api.ClusterConfig) error {
	err := runParallelOnHosts(conf.Spec.Hosts, conf, investigateHost)
	if err != nil {
		return err
	}
	// Gather UCP related facts
	conf.Spec.Ucp.Metadata = &api.UcpMetadata{
		ClusterID:        "",
		Installed:        false,
		InstalledVersion: "",
	}
	swarmLeader := conf.Spec.SwarmLeader()
	// If engine is installed, we can collect some UCP & Swarm related info too
	if swarmLeader.Metadata.EngineVersion != "" {
		ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
		}
		conf.Spec.Ucp.Metadata = ucpMeta
		if ucpMeta.Installed {
			log.Infof("%s: UCP has version %s", swarmLeader.Address, ucpMeta.InstalledVersion)
		} else {
			log.Infof("%s: UCP is not installed", swarmLeader.Address)
		}
		conf.Spec.Ucp.Metadata.ClusterID = swarm.ClusterID(swarmLeader)
	}

	return nil
}

func investigateHost(h *api.Host, c *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h.Address)

	os := &api.OsRelease{}
	if isWindows(h) {
		h.Connection.SetWindows(true)
		winOs, err := ResolveWindowsOsRelease(h)
		if err != nil {
			return err
		}
		os = winOs
	} else {
		linuxOs, err := ResolveLinuxOsRelease(h)
		if err != nil {
			return err
		}
		os = linuxOs
	}

	h.Metadata = &api.HostMetadata{
		Os: os,
	}
	err := api.ResolveHostConfigurer(h)
	if err != nil {
		return err
	}

	// TODO move this into the host validations
	testfn := "launchpad_connection_test.txt"
	log.Debugf("%s: testing connection", h.Address)

	// cleanup
	if h.Configurer.FileExist(testfn) {
		if err := h.Configurer.DeleteFile(testfn); err != nil {
			return err
		}
	}

	if err := h.Configurer.WriteFile(testfn, "test", "0600"); err != nil {
		return err
	}

	if !h.Configurer.FileExist(testfn) {
		return fmt.Errorf("connection file write test failed at file exist check")
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

	h.Metadata.Hostname = h.Configurer.ResolveHostname()
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

func isWindows(h *api.Host) bool {
	err := h.ExecCmd(`powershell -Command "Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\""`, "", false, false)
	if err != nil {
		return false
	}
	return true
}

// ResolveWindowsOsRelease ...
func ResolveWindowsOsRelease(h *api.Host) (*api.OsRelease, error) {
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
func ResolveLinuxOsRelease(h *api.Host) (*api.OsRelease, error) {
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
