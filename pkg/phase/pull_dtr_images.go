package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/docker"
	log "github.com/sirupsen/logrus"
)

// PullDTRImages phase implementation
type PullDTRImages struct {
	Analytics
	DtrPhase
}

// Title for the phase
func (p *PullDTRImages) Title() string {
	return "Pull DTR images"
}

// Run pulls images in parallel across nodes via a workerpool of 5
func (p *PullDTRImages) Run() error {
	images, err := p.ListImages()
	if err != nil {
		return err
	}
	log.Debugf("loaded DTR images list: %v", images)

	imageRepo := p.config.Spec.Dtr.ImageRepo
	if api.IsCustomImageRepo(imageRepo) {
		pullList := docker.AllToRepository(images, imageRepo)
		// In case of custom image repo, we need to pull and retag all the images on all DTR hosts
		return runParallelOnHosts(p.config.Spec.Dtrs(), p.config, func(h *api.Host, c *api.ClusterConfig) error {
			if err := docker.PullImages(h, pullList); err != nil {
				return err
			}
			return docker.RetagAllToRepository(h, pullList, images[0].Repository)
		})
	}

	return docker.PullImages(p.config.Spec.DtrLeader(), images)
}

// ListImages obtains a list of images from DTR
func (p *PullDTRImages) ListImages() ([]*docker.Image, error) {
	dtrLeader := p.config.Spec.DtrLeader()
	bootstrap := docker.NewImage(p.config.Spec.Dtr.GetBootstrapperImage())

	if !bootstrap.Exist(dtrLeader) {
		if err := bootstrap.Pull(dtrLeader); err != nil {
			return []*docker.Image{}, err
		}
	}
	output, err := dtrLeader.ExecWithOutput(dtrLeader.Configurer.DockerCommandf("run --rm %s images", bootstrap))
	if err != nil {
		return []*docker.Image{}, fmt.Errorf("%s: failed to get DTR image list", dtrLeader)
	}

	return docker.AllFromString(output), nil
}
