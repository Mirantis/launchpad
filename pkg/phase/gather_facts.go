package phase

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/host"
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

func investigateHost(h *host.Host, wg *sync.WaitGroup) {
	defer wg.Done()

	h.Metadata = &host.HostMetadata{
		Hostname:        resolveHostname(h),
		InternalAddress: resolveInternalIP(h),
	}

	log.Debugf("host %s has internal address: %s", h.Address, h.Metadata.InternalAddress)
}

func resolveHostname(h *host.Host) string {
	hostname, _ := h.ExecWithOutput("hostname -s")

	return hostname
}

func resolveInternalIP(h *host.Host) string {
	output, _ := h.ExecWithOutput(fmt.Sprintf("ip -o addr show dev %s scope global", h.PrivateInterface))
	lines := strings.Split(output, "\r\n")
	for _, line := range lines {
		items := strings.Fields(line)
		addrItems := strings.Split(items[3], "/")
		if addrItems[0] != h.Address {
			return addrItems[0]
		}
	}
	return h.Address
}
