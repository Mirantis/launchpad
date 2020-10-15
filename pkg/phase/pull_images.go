package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// PullImages phase implementation
type PullImages struct {
	Analytics
	BasicPhase
	Dtr bool
}

// Title for the phase
func (p *PullImages) Title() string {
	if p.Dtr {
		return "Pull DTR images"
	}
	return "Pull UCP images"
}

// Run pulls images in parallel across nodes via a workerpool of 5
func (p *PullImages) Run() error {
	imageRepo := p.config.Spec.Ucp.ImageRepo
	product := "UCP"
	if p.Dtr {
		imageRepo = p.config.Spec.Dtr.ImageRepo
		product = "DTR"
	}

	err := runParallelOnHosts(p.config.Spec.Hosts, p.config, func(h *api.Host, c *api.ClusterConfig) error {
		return h.AuthenticateDocker(imageRepo)
	})

	if err != nil {
		return err
	}
	images, err := p.ListImages(p.config)
	if err != nil {
		return NewError(err.Error())
	}
	log.Debugf("loaded %s images list: %v", product, images)

	if api.IsCustomImageRepo(imageRepo) {
		// Store map of original --> custom repo images
		imageMap := make(map[string]string, len(images))
		for index, i := range images {
			newImage := p.ImageFromCustomRepo(i, imageRepo)
			imageMap[i] = newImage
			images[index] = newImage
		}
		// In case of custom image repo, we need to pull and retag all the images on all hosts
		return runParallelOnHosts(p.config.Spec.Hosts, p.config, func(h *api.Host, c *api.ClusterConfig) error {
			err := ImagePull(h, images)
			if err != nil {
				return err
			}
			return RetagImages(h, imageMap)
		})
	}
	// Normally we pull only on managers, let workers pull needed stuff on-demand
	return runParallelOnHosts(p.config.Spec.Managers(), p.config, func(h *api.Host, c *api.ClusterConfig) error {
		return ImagePull(h, images)
	})
}

// ImageFromCustomRepo will replace the organization part in an image name
func (p *PullImages) ImageFromCustomRepo(image, repo string) string {
	return fmt.Sprintf("%s%s", repo, image[strings.IndexByte(image, '/'):])
}

// ListImages obtains a list of images depending on which product is being
// listed
func (p *PullImages) ListImages(config *api.ClusterConfig) ([]string, error) {
	manager := config.Spec.SwarmLeader()
	imageFlag := "--list"
	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	product := "UCP"
	if p.Dtr {
		product = "DTR"
		// Use one of the DTRs to obtain the image list since we haven't
		// yet established a "leader" when this runs
		manager = config.Spec.DtrLeader()
		image = fmt.Sprintf("%s/dtr:%s", config.Spec.Dtr.ImageRepo, config.Spec.Dtr.Version)
		imageFlag = ""
	}

	if !manager.ImageExist(image) {
		err := manager.PullImage(image)
		if err != nil {
			return []string{}, err
		}
	}
	output, err := manager.ExecWithOutput(manager.Configurer.DockerCommandf("run --rm %s images %s", image, imageFlag))
	if err != nil {
		return []string{}, fmt.Errorf("failed to get %s image list", product)
	}

	return strings.Split(output, "\n"), nil
}

// ImagePull pulls images on a host in parallel using a workerpool with 5
// workers. Essentially we pull 5 images in parallel.
func ImagePull(h *api.Host, images []string) error {
	wp := workerpool.New(5)
	defer wp.StopWait()

	for _, image := range images {
		i := image // So we can safely pass i forward to pool without it getting mutated
		wp.Submit(func() {
			retry.Do(
				func() error {
					if !h.ImageExist(i) {
						log.Infof("%s: pulling image %s", h, i)
						return h.PullImage(i)
					}
					log.Infof("%s: image %s already exists", h, i)
					return nil
				},
				retry.RetryIf(func(err error) bool {
					return !(strings.Contains(err.Error(), "pull access") || strings.Contains(err.Error(), "manifest unknown"))
				}),
				retry.OnRetry(func(n uint, err error) {
					if err != nil {
						log.Warnf("%s: failed to pull image %s - retrying", h, i)
					}
				}),
				retry.Attempts(2),
			)
		})
	}
	return nil
}

// RetagImages takes a list of images and retags them for use with a custom
// image repo
func RetagImages(h *api.Host, imageMap map[string]string) error {
	for dockerImage, realImage := range imageMap {
		retagCmd := h.Configurer.DockerCommandf("tag %s %s", realImage, dockerImage)
		log.Debugf("%s: retag %s --> %s", h, realImage, dockerImage)
		err := h.Exec(retagCmd)
		if err != nil {
			return err
		}
	}

	return nil
}
