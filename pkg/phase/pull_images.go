package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

type PullImages struct{}

func (p *PullImages) Title() string {
	return "Pull images"
}

func (p *PullImages) Run(c *config.ClusterConfig) *PhaseError {
	images, err := p.listImages(c)
	if err != nil {
		return NewPhaseError(err.Error())
	}
	log.Debugf("loaded images list: %v", images)

	return runParallelOnHosts(c.Controllers(), c, func(h *config.Host, c *config.ClusterConfig) error {
		return p.pullImages(h, images)
	})
}

func (p *PullImages) listImages(config *config.ClusterConfig) ([]string, error) {
	controller := config.Controllers()[0]
	image := fmt.Sprintf("%s/ucp:%s", config.Ucp.ImageRepo, config.Ucp.Version)
	err := controller.PullImage(image)
	if err != nil {
		return []string{}, err
	}
	output, err := controller.ExecWithOutput(fmt.Sprintf("sudo docker run --rm %s images --list", image))
	if err != nil {
		return []string{}, fmt.Errorf("failed to get image list")
	}

	return strings.Split(output, "\n"), nil
}

// Pulls images on a host in parallel using a workerpool with 5 workers. Essentially we pull 5 images in parallel.
func (p *PullImages) pullImages(host *config.Host, images []string) error {
	wp := workerpool.New(5)
	defer wp.StopWait()

	for _, image := range images {
		i := image // So we can safely pass i forward to pool without it getting mutated
		wp.Submit(func() {
			log.Debugf("%s: pulling image %s", host.Address, i)
			e := host.PullImage(i)
			if e != nil {
				log.Warnf("%s: failed to pull image %s", host.Address, i)
			}
		})
	}
	return nil
}
