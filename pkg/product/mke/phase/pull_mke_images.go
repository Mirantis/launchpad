package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/docker"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// PullMKEImages phase implementation
type PullMKEImages struct {
	phase.Analytics
	phase.BasicPhase
	phase.CleanupDisabling
}

// Title for the phase
func (p *PullMKEImages) Title() string {
	return "Pull MKE images"
}

// Run pulls images in parallel across nodes via a workerpool of 5
func (p *PullMKEImages) Run() error {
	images, err := p.ListImages()
	if err != nil {
		return err
	}
	log.Debugf("loaded images list: %v", images)

	imageRepo := p.Config.Spec.MKE.ImageRepo
	if api.IsCustomImageRepo(imageRepo) {
		pullList := docker.AllToRepository(images, imageRepo)
		// In case of custom image repo, we need to pull and retag all the images on all hosts
		return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, func(h *api.Host, c *api.ClusterConfig) error {
			if err := docker.PullImages(h, pullList); err != nil {
				return err
			}
			return docker.RetagAllToRepository(h, pullList, images[0].Repository)
		})
	}

	// Normally we pull only on managers, let workers pull needed stuff on-demand
	return phase.RunParallelOnHosts(p.Config.Spec.Managers(), p.Config, func(h *api.Host, c *api.ClusterConfig) error {
		return docker.PullImages(h, images)
	})
}

// ListImages obtains a list of images from MKE
func (p *PullMKEImages) ListImages() ([]*docker.Image, error) {
	manager := p.Config.Spec.SwarmLeader()
	bootstrap := docker.NewImage(p.Config.Spec.MKE.GetBootstrapperImage())

	if !bootstrap.Exist(manager) {
		if err := bootstrap.Pull(manager); err != nil {
			return []*docker.Image{}, err
		}
	}

	runFlags := common.Flags{"-v /var/run/docker.sock:/var/run/docker.sock"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	output, err := manager.ExecOutput(manager.Configurer.DockerCommandf("run %s %s images --list", runFlags.Join(), bootstrap))
	if err != nil {
		return []*docker.Image{}, fmt.Errorf("%s: failed to get MKE image list", manager)
	}

	return docker.AllFromString(output), nil
}
