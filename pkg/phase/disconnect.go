package phase

import (
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/host"
	"github.com/sirupsen/logrus"
)

type Disconnect struct{}

func (p *Disconnect) Title() string {
	return "Close SSH Connection"
}

func (p *Disconnect) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Hosts {
		wg.Add(1)
		go p.disconnectHost(host, &wg)
	}
	wg.Wait()

	return nil
}

func (p *Disconnect) disconnectHost(host *host.Host, wg *sync.WaitGroup) error {
	defer wg.Done()
	host.Connect()
	logrus.Printf("%s: SSH connection closed", host.Address)
	return nil
}
