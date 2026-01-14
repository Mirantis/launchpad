package docker

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// regex for pulling images names from command string output, common for msr/mke bootstrappers.
var imageListFromOutputString = regexp.MustCompile(`(?m)^(?<repo>[\w\.-]*)(?<port>:[0-9]*)?(?<path>[\w-\/]*)?\/(?<name>[\w-]+)\:(?<tag>[\w\.-]+)[" "\t]*$`)

// Image describes a docker image.
type Image struct {
	Repository string
	Name       string
	Tag        string
}

// NewImage parses an image URL and returns a new Image instance.
func NewImage(s string) *Image {
	repo := s[:strings.LastIndexByte(s, '/')]
	name := s[strings.LastIndexByte(s, '/')+1 : strings.LastIndexByte(s, ':')]
	tag := s[strings.LastIndexByte(s, ':')+1:]
	return &Image{Repository: repo, Name: name, Tag: tag}
}

// AllFromString parses a list of images from a newline separated string and returns a list of images.
func AllFromString(s string) (list []*Image) {
	for _, match := range imageListFromOutputString.FindAllStringSubmatch(s, -1) {
		list = append(list, &Image{
			Repository: strings.Join([]string{match[1], match[2], match[3]}, ""),
			Name:       match[4],
			Tag:        match[5],
		})
	}
	return list
}

func (i *Image) String() string {
	return fmt.Sprintf("%s/%s:%s", i.Repository, i.Name, i.Tag)
}

// Pull pulls an image on a host.
func (i *Image) Pull(h *mkeconfig.Host) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: pulling image %s", h, i)
			if i.Exist(h) {
				log.Infof("%s: already exists: %s", h, i)
				return nil
			}
			output, err := h.ExecOutput(h.Configurer.DockerCommandf("pull %s", i))
			if err != nil {
				return fmt.Errorf("%s: failed to pull image: %s: %w", h, output, err)
			}
			return nil
		},
		retry.RetryIf(func(err error) bool {
			return !strings.Contains(err.Error(), "pull access") && !strings.Contains(err.Error(), "manifest unknown")
		}),
		retry.OnRetry(func(_ uint, err error) {
			if err != nil {
				log.Warnf("%s: failed to pull image %s - retrying", h, i)
			}
		}),
		retry.Attempts(2),
	)
	if err != nil {
		return fmt.Errorf("retry count exceeded: %w", err)
	}
	return nil
}

// Retag retags image A to image B.
func (i *Image) Retag(h *mkeconfig.Host, a, b *Image) error {
	log.Debugf("%s: retag %s --> %s", h, a, b)
	if err := h.Exec(h.Configurer.DockerCommandf("tag %s %s", a, b)); err != nil {
		return fmt.Errorf("%s: failed to retag image %s --> %s: %w", h, a, b, err)
	}
	return nil
}

// Exist returns true if a docker image exists on the host.
func (i *Image) Exist(h *mkeconfig.Host) bool {
	return h.Exec(h.Configurer.DockerCommandf("image inspect %s --format '{{.ID}}'", i)) == nil
}

// PullImages pulls multiple images parallelly by using a worker pool.
func PullImages(h *mkeconfig.Host, images []*Image) error {
	wp := workerpool.New(5)
	defer wp.StopWait()

	var mutex sync.Mutex
	var lastError error

	for _, image := range images {
		i := image // So we can safely pass i forward to pool without it getting mutated
		wp.Submit(func() {
			mutex.Lock()
			defer mutex.Unlock()
			if lastError != nil {
				return
			}

			err := i.Pull(h)
			if err != nil {
				mutex.Lock()
				lastError = err
			}
		})
	}

	return lastError
}

// RetagAllToRepository retags all images in a list to another repository.
func RetagAllToRepository(h *mkeconfig.Host, images []*Image, repo string) error {
	for _, i := range images {
		newImage := &Image{
			Repository: repo,
			Name:       i.Name,
			Tag:        i.Tag,
		}
		if err := i.Retag(h, i, newImage); err != nil {
			return err
		}
	}

	return nil
}

// AllToRepository generates a new list with a different repository.
func AllToRepository(images []*Image, repo string) (list []*Image) {
	for _, i := range images {
		list = append(list, &Image{
			Repository: repo,
			Name:       i.Name,
			Tag:        i.Tag,
		})
	}
	return list
}

var errInvalidVersion = errors.New("invalid image version")

// ImageRepoAndTag returns the Repo and tag from a container image.
//
//	e.g. `dtr.efzp.com:9026/mirantis/ucp-agent:3.8.10` => `dtr.efzp.com:9026/mirantis/ucp-agent`, `3.8.10`
func ImageRepoAndTag(image string) (string, string, error) {
	vparts := strings.Split(image, ":")
	vpartslen := len(vparts)
	if vpartslen < 2 || vpartslen > 3 {
		return "", "", fmt.Errorf("%w: malformed version output: %s", errInvalidVersion, image)
	}

	repo := strings.Join(vparts[0:vpartslen-1], ":")
	version := vparts[vpartslen-1]
	return repo, version, nil
}
