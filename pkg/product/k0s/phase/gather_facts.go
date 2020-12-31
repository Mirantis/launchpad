package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"

	"github.com/Mirantis/mcc/pkg/product/k0s/api"

	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/oracle"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/sles"
	"github.com/cobaugh/osrelease"
	log "github.com/sirupsen/logrus"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts
type GatherFacts struct {
	phase.Analytics
	BasicPhase
}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run() error {
	err := RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.investigateHost)
	if err != nil {
		return err
	}

	return nil
}

func (p *GatherFacts) investigateHost(h *api.Host, c *api.ClusterConfig) error {
	if h.Connection == nil {
		return fmt.Errorf("%s: not connected", h)
	}

	log.Infof("%s: gathering host facts", h)

	os := &common.OsRelease{}

	if p.isWindows(h) {
		h.Connection.SetWindows(true)
		winOs, err := p.resolveWindowsOsRelease(h)
		if err != nil {
			return err
		}
		os = winOs
	} else {
		h.Connection.SetWindows(false)
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

	version, err := h.K0sVersion()
	if err != nil {
		version = "not installed"
		log.Infof("%s: K0s not installed", h)
	} else {
		log.Infof("%s: is running k0s version %s", h, version)
	}

	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.LongHostname = h.Configurer.ResolveLongHostname()
	h.Metadata.K0sVersion = version

	log.Infof("%s: is running \"%s\"", h, h.Metadata.Os.Name)
	log.Infof("%s: internal address: %s", h, h.Metadata.InternalAddress)

	log.Infof("%s: gathered all facts", h)

	return nil
}

func (p *GatherFacts) isWindows(h *api.Host) bool {
	return h.Exec("cmd /c exit 0") == nil
}

// ResolveWindowsOsRelease ... TODO: this implementation belongs somewhere else
func (p *GatherFacts) resolveWindowsOsRelease(h *api.Host) (*common.OsRelease, error) {
	osName, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").ProductName"`)
	osMajor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMajorVersionNumber"`)
	osMinor, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMinorVersionNumber"`)
	osBuild, _ := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentBuild"`)

	version := fmt.Sprintf("%s.%s.%s", osMajor, osMinor, osBuild)
	osRelease := &common.OsRelease{
		ID:      fmt.Sprintf("windows-%s", version),
		Name:    osName,
		Version: version,
	}

	return osRelease, nil
}

// ResolveLinuxOsRelease ...
func (p *GatherFacts) resolveLinuxOsRelease(h *api.Host) (*common.OsRelease, error) {
	output, err := h.ExecWithOutput("cat /etc/os-release")
	if err != nil {
		return nil, err
	}
	info, err := osrelease.ReadString(output)
	if err != nil {
		return nil, err
	}
	osRelease := &common.OsRelease{
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
