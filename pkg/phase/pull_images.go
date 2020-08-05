package phase

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// PullImages phase implementation
type PullImages struct {
	Analytics
}

// Title for the phase
func (p *PullImages) Title() string {
	return "Pull images"
}

// Run pulls all the needed images on managers in parallel.
// Parallel on each host and pulls 5 images at a time on each host.
func (p *PullImages) Run(c *api.ClusterConfig) error {
	err := runParallelOnHosts(c.Spec.Hosts, c, func(h *api.Host, c *api.ClusterConfig) error {
		return h.AuthenticateDocker(c.Spec.Ucp.ImageRepo)
	})

	if err != nil {
		return err
	}
	images, err := p.listImages(c)
	if err != nil {
		return NewError(err.Error())
	}
	log.Debugf("loaded images list: %v", images)

	if c.Spec.Ucp.IsCustomImageRepo() {
		// Store map of original --> custom repo images
		imageMap := make(map[string]string, len(images))
		for index, i := range images {
			newImage := p.ImageFromCustomRepo(i, c.Spec.Ucp.ImageRepo)
			imageMap[i] = newImage
			images[index] = newImage
		}
		// In case of custom image repo, we need to pull and retag all the images on all hosts
		return runParallelOnHosts(c.Spec.Hosts, c, func(h *api.Host, c *api.ClusterConfig) error {
			err := p.pullImages(h, images)
			if err != nil {
				return err
			}
			return p.retagImages(h, imageMap)
		})
	}
	// Normally we pull only on managers, let workers pull needed stuff on-demand
	return runParallelOnHosts(c.Spec.Managers(), c, func(h *api.Host, c *api.ClusterConfig) error {
		return p.pullImages(h, images)
	})
}

func (p *PullImages) ImageFromCustomRepo(image, repo string) string {
	return fmt.Sprintf("%s%s", repo, image[strings.IndexByte(image, '/'):])
}

func (p *PullImages) listImages(config *api.ClusterConfig) ([]string, error) {
	manager := config.Spec.SwarmLeader()

	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	err := manager.PullImage(image)
	if err != nil {
		return []string{}, err
	}
	output, err := manager.ExecWithOutput(manager.Configurer.DockerCommandf("run --rm %s images --list", image))
	if err != nil {
		return []string{}, fmt.Errorf("failed to get image list")
	}

	return strings.Split(output, "\n"), nil
}

// Pulls images on a host in parallel using a workerpool with 5 workers. Essentially we pull 5 images in parallel.
func (p *PullImages) pullImages(host *api.Host, images []string) error {
	wp := workerpool.New(5)
	defer wp.StopWait()

	for _, image := range images {
		i := image // So we can safely pass i forward to pool without it getting mutated
		wp.Submit(func() {
			log.Infof("%s: pulling image %s", host.Address, i)
			e := host.PullImage(i)
			if e != nil {
				log.Warnf("%s: failed to pull image %s", host.Address, i)
			}
		})
	}
	return nil
}

func (p *PullImages) retagImages(host *api.Host, imageMap map[string]string) error {

	for dockerImage, realImage := range imageMap {
		retagCmd := host.Configurer.DockerCommandf("tag %s %s", realImage, dockerImage)
		log.Debugf("%s: retag %s --> %s", host.Address, realImage, dockerImage)
		_, err := host.ExecWithOutput(retagCmd)
		if err != nil {
			return err
		}
	}

	return nil
}
