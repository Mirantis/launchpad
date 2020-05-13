package phase

import (
	"fmt"
	"sync"

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

func (p *GatherHostFacts) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Hosts {
		wg.Add(1)
		log.Infof("gathering host %s facts", host.Address)
		go investigateHost(host, &wg)
	}
	wg.Wait()

	return nil
}

func investigateHost(h *config.Host, wg *sync.WaitGroup) {
	defer wg.Done()

	os, err := resolveOsRelease(h)
	if err != nil {
		return
	}
	h.Metadata = &config.HostMetadata{
		Os: os,
	}
	err = resolveHostConfigurer(h)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.InternalAddress = h.Configurer.ResolveInternalIP()

	log.Debugf("host %s has internal address: %s", h.Address, h.Metadata.InternalAddress)
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
