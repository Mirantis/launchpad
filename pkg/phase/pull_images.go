package phase

import (
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/host"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

type PullImages struct{}

// FIXME Version needs to come from config
var Images = []string{"docker/ucp:3.3.0-rc1"}

func (p *PullImages) Title() string {
	return "Pull images"
}

func (p *PullImages) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Controllers() {
		wg.Add(1)
		go p.pullImages(host, &wg)
	}
	wg.Wait()

	return nil
}

func (p *PullImages) pullImages(host *host.Host, wg *sync.WaitGroup) error {
	defer wg.Done()
	for _, image := range Images {
		err := retry.Do(
			func() error {
				log.Debugf("Starting to pull image: %s", image)
				err := host.PullImage(image)
				if err == nil {
					log.Infof("Pulled %s sucesfully", image)
				}
				return err
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
