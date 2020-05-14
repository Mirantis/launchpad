package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/config"
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	"github.com/cobaugh/osrelease"
	log "github.com/sirupsen/logrus"
)

type GatherHostFacts struct{}

func (p *GatherHostFacts) Title() string {
	return "Gather Host Facts"
}

func (p *GatherHostFacts) Run(config *config.ClusterConfig) *PhaseError {
	return runParallelOnHosts(config.Hosts, config, investigateHost)
}

func investigateHost(h *config.Host, c *config.ClusterConfig) error {
	log.Infof("gathering host %s facts", h.Address)

	os, err := resolveOsRelease(h)
	if err != nil {
		return err
	}
	h.Metadata = &config.HostMetadata{
		Os: os,
	}
	err = resolveHostConfigurer(h)
	if err != nil {
		return err
	}
	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.InternalAddress = h.Configurer.ResolveInternalIP()

	log.Debugf("host %s has internal address: %s", h.Address, h.Metadata.InternalAddress)

	return nil
}

func resolveOsRelease(h *config.Host) (*config.OsRelease, error) {
	err := h.Exec("uname | grep -q -i linux")
	if err != nil {
		return nil, err
	}
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
