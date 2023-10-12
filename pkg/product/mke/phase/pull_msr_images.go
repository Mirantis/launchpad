package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/docker"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// PullMSRImages phase implementation.
type PullMSRImages struct {
	phase.Analytics
	MSRPhase
}

// Title for the phase.
func (p *PullMSRImages) Title() string {
	return "Pull MSR images"
}

// Run pulls images in parallel across nodes via a workerpool of 5.
func (p *PullMSRImages) Run() error {
	images, err := p.ListImages()
	if err != nil {
		return err
	}
	log.Debugf("loaded MSR images list: %v", images)

	imageRepo := p.Config.Spec.MSR.ImageRepo
	if api.IsCustomImageRepo(imageRepo) {
		pullList := docker.AllToRepository(images, imageRepo)
		// In case of custom image repo, we need to pull and retag all the images on all MSR hosts
		return phase.RunParallelOnHosts(p.Config.Spec.MSRs(), p.Config, func(h *api.Host, c *api.ClusterConfig) error {
			if err := docker.PullImages(h, pullList); err != nil {
				return err
			}
			return docker.RetagAllToRepository(h, pullList, images[0].Repository)
		})
	}

	return docker.PullImages(p.Config.Spec.MSRLeader(), images)
}

// ListImages obtains a list of images from MSR.
func (p *PullMSRImages) ListImages() ([]*docker.Image, error) {
	msrLeader := p.Config.Spec.MSRLeader()
	bootstrap := docker.NewImage(p.Config.Spec.MSR.GetBootstrapperImage())

	if !bootstrap.Exist(msrLeader) {
		if err := bootstrap.Pull(msrLeader); err != nil {
			return []*docker.Image{}, err
		}
	}
	output, err := msrLeader.ExecOutput(msrLeader.Configurer.DockerCommandf("run --rm %s images", bootstrap))
	if err != nil {
		return []*docker.Image{}, fmt.Errorf("%s: failed to get MSR image list", msrLeader)
	}

	return docker.AllFromString(output), nil
}
