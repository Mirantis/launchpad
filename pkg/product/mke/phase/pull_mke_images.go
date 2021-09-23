package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/docker"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/hashicorp/go-version"

	log "github.com/sirupsen/logrus"
)

// PullMKEImages phase implementation
type PullMKEImages struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *PullMKEImages) Title() string {
	return "Pull MKE images"
}

// Run pulls images in parallel across nodes via a workerpool of 5
func (p *PullMKEImages) Run() error {
	images, err := p.ListImages(false)
	if err != nil {
		return err
	}
	log.Debugf("loaded linux images list: %v", images)

	var winImages []*docker.Image
	var winHosts api.Hosts = p.Config.Spec.Hosts.Filter(func(h *api.Host) bool { return h.IsWindows() })

	if len(winHosts) > 0 {
		winImages, err = p.ListImages(true)
		if err != nil {
			return err
		}
		log.Debugf("loaded windows images list: %v", winImages)
	}

	imageRepo := p.Config.Spec.MKE.ImageRepo

	if api.IsCustomImageRepo(imageRepo) {
		pullList := docker.AllToRepository(images, imageRepo)
		pullListWin := docker.AllToRepository(winImages, imageRepo)
		return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, func(h *api.Host, c *api.ClusterConfig) error {
			var err error
			var list []*docker.Image

			if h.IsWindows() {
				list = pullListWin
			} else {
				list = pullList
			}

			if err = docker.PullImages(h, list); err != nil {
				return err
			}

			mkeV, err := version.NewVersion(p.Config.Spec.MKE.Version)
			if err != nil {
				return err
			}
			modV, err := version.NewVersion("3.3.7")

			// MKE 3.3.8 introduced a change / bug / feature that makes it only look at images from "mirantis" repository.
			// Before that, MKE was looking for images from the same repository as the bootstrapper. This version switch logic
			// should make it work across MKE versions.
			var retagRepo string
			if mke.VersionGreaterThan(mkeV, modV) {
				retagRepo = "mirantis"
			} else {
				retagRepo = images[0].Repository
			}

			log.Debugf("%s: retagging images to %s", h, retagRepo)

			return docker.RetagAllToRepository(h, list, retagRepo)
		})
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Managers(), p.Config, func(h *api.Host, c *api.ClusterConfig) error {
		log.Infof("%s: pulling linux images", h)
		return docker.PullImages(h, images)
	})

	if err != nil {
		return err
	}

	if len(winHosts) > 0 {
		return phase.RunParallelOnHosts(winHosts, p.Config, func(h *api.Host, c *api.ClusterConfig) error {
			log.Infof("%s: pulling windows images", h)
			return docker.PullImages(h, winImages)
		})
	}

	return nil
}

// ListImages obtains a list of images from MKE
func (p *PullMKEImages) ListImages(win bool) ([]*docker.Image, error) {
	manager := p.Config.Spec.SwarmLeader()
	bootstrap := docker.NewImage(p.Config.Spec.MKE.GetBootstrapperImage())

	if !bootstrap.Exist(manager) {
		if err := bootstrap.Pull(manager); err != nil {
			return []*docker.Image{}, err
		}
	}

	runFlags := common.Flags{"--rm", "-v /var/run/docker.sock:/var/run/docker.sock"}

	if manager.Configurer.SELinuxEnabled(manager) {
		runFlags.Add("--security-opt label=disable")
	}

	imageFlags := common.Flags{"--list"}

	if win {
		imageFlags.Add("--enable-windows")
	}

	output, err := manager.ExecOutput(manager.Configurer.DockerCommandf("run %s %s images %s", runFlags.Join(), bootstrap, imageFlags.Join()))
	if err != nil {
		return []*docker.Image{}, fmt.Errorf("%s: failed to get MKE image list", manager)
	}

	return docker.AllFromString(output), nil
}
