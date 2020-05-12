package phase

import (
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/host"
	retry "github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct{}

func (p *Connect) Title() string {
	return "Open SSH Connection"
}

func (p *Connect) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Hosts {
		wg.Add(1)
		go p.connectHost(host, &wg)
	}
	wg.Wait()

	return nil
}

func (p *Connect) connectHost(host *host.Host, wg *sync.WaitGroup) error {
	host.Normalize() // FIXME we need to handle this better somewhere else...
	defer wg.Done()
	err := retry.Do(
		func() error {
			logrus.Infof("%s: opening SSH connection", host.Address)
			err := host.Connect()
			if err != nil {
				logrus.Errorf("%s: failed to connect -> %s", host.Address, err.Error())
			}
			return err
		},
	)
	if err != nil {
		logrus.Errorf("%s: failed to open connection", host.Address)
		return err
	}

	logrus.Printf("%s: SSH connection opened", host.Address)
	return nil
}
