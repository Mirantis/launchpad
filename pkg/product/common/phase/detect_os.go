package phase

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	// needed to load the build func in package init

	"github.com/Mirantis/mcc/pkg/exec"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	log "github.com/sirupsen/logrus"

	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/oracle"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/sles"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"

	"github.com/cobaugh/osrelease"
)

type execableHost interface {
	String() string
	Exec(cmd string, opts ...exec.Option) error
	ExecWithOutput(cmd string, opts ...exec.Option) (string, error)
	ResolveHostConfigurer() error
	IsWindows() bool
}

// DetectOS phase implementation to collect facts (OS, version etc.) from hosts
type DetectOS struct {
	hosts []execableHost
}

// Prepare collects and casts the hosts from the config
func (p *DetectOS) Prepare(config interface{}) error {
	r := reflect.ValueOf(config).Elem()
	spec := r.FieldByName("Spec").Elem()
	hosts := spec.FieldByName("Hosts")
	for i := 0; i < hosts.Len(); i++ {
		if h, ok := hosts.Index(i).Interface().(execableHost); ok {
			p.hosts = append(p.hosts, h)
		}
	}

	return nil
}

// ShouldRun is true when there are hosts
func (p *DetectOS) ShouldRun() bool {
	return len(p.hosts) > 0
}

// CleanUp does nothing
func (p *DetectOS) CleanUp() {}

// Title for the phase
func (p *DetectOS) Title() string {
	return "Detect Operating Systems"
}

// Run collect all the facts from hosts in parallel
func (p *DetectOS) Run() error {
	var wg sync.WaitGroup
	var errors []string
	type erritem struct {
		host string
		err  error
	}
	ec := make(chan erritem, 1)

	wg.Add(len(p.hosts))

	for _, h := range p.hosts {
		go func(h execableHost) {
			ec <- erritem{h.String(), p.detectOS(h)}
		}(h)
	}

	go func() {
		for e := range ec {
			if e.err != nil {
				errors = append(errors, fmt.Sprintf("%s: %s", e.host, e.err.Error()))
			}
			wg.Done()
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("failed on %d hosts:\n - %s", len(errors), strings.Join(errors, "\n - "))
	}

	return nil
}

func (p *DetectOS) detectOS(h execableHost) error {
	log.Infof("%s: detecting host OS", h)

	var os *common.OsRelease
	if h.IsWindows() {
		log.Infof("%s: resolving windows revision", h)
		winOs, err := p.resolveWindowsOsRelease(h)
		if err != nil {
			return err
		}
		os = winOs
	} else {
		log.Infof("%s: resolving distribution", h)
		linuxOs, err := p.resolveLinuxOsRelease(h)
		if err != nil {
			return err
		}
		os = linuxOs
	}

	r := reflect.ValueOf(h).Elem()
	meta := r.FieldByName("Metadata").Elem()
	osMeta := meta.FieldByName("Os")
	osMeta.SetPointer(unsafe.Pointer(os))
	if err := h.ResolveHostConfigurer(); err != nil {
		return err
	}

	return nil
}

func (p *DetectOS) resolveWindowsOsRelease(h execableHost) (*common.OsRelease, error) {
	osName, err := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").ProductName"`)
	if err != nil {
		return nil, err
	}
	osMajor, err := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMajorVersionNumber"`)
	if err != nil {
		return nil, err
	}
	osMinor, err := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentMinorVersionNumber"`)
	if err != nil {
		return nil, err
	}
	osBuild, err := h.ExecWithOutput(`powershell -Command "(Get-ItemProperty \"HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\").CurrentBuild"`)
	if err != nil {
		return nil, err
	}

	version := fmt.Sprintf("%s.%s.%s", osMajor, osMinor, osBuild)
	osRelease := &common.OsRelease{
		ID:      fmt.Sprintf("windows-%s", version),
		Name:    osName,
		Version: version,
	}

	return osRelease, nil
}

// ResolveLinuxOsRelease ...
func (p *DetectOS) resolveLinuxOsRelease(h execableHost) (*common.OsRelease, error) {
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
