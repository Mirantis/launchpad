package docker

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Mirantis/mcc/pkg/product/mke/api"
	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// Image describes a docker image
type Image struct {
	Repository string
	Name       string
	Tag        string
}

// NewImage parses an image URL and returns a new Image instance
func NewImage(s string) *Image {
	repo := s[:strings.LastIndexByte(s, '/')]
	name := s[strings.LastIndexByte(s, '/')+1 : strings.LastIndexByte(s, ':')]
	tag := s[strings.LastIndexByte(s, ':')+1:]
	return &Image{Repository: repo, Name: name, Tag: tag}
}

// AllFromString parses a list of images from a newline separated string and returns a list of images
func AllFromString(s string) (list []*Image) {
	re := regexp.MustCompile("(?m)^(?P<repo>.*)/(?P<name>.+?):(?P<tag>.+?)$")
	for _, match := range re.FindAllStringSubmatch(s, -1) {
		list = append(list, &Image{
			Repository: match[1],
			Name:       match[2],
			Tag:        match[3],
		})
	}
	return list
}

func (i *Image) String() string {
	return fmt.Sprintf("%s/%s:%s", i.Repository, i.Name, i.Tag)
}

// Pull pulls an image on a host
func (i *Image) Pull(h *api.Host) error {
	return retry.Do(
		func() error {
			log.Infof("%s: pulling image %s", h, i)
			if i.Exist(h) {
				log.Infof("%s: already exists: %s", h, i)
				return nil
			}
			output, err := h.ExecWithOutput(h.Configurer.DockerCommandf("pull %s", i))
			if err != nil {
				return fmt.Errorf("%s: failed to pull image: %s", h, output)
			}
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
}

// Retag retags image A to image B
func (i *Image) Retag(h *api.Host, a, b *Image) error {
	log.Debugf("%s: retag %s --> %s", h, a, b)
	return h.Exec(h.Configurer.DockerCommandf("tag %s %s", a, b))
}

// Exist returns true if a docker image exists on the host
func (i *Image) Exist(h *api.Host) bool {
	return h.Exec(h.Configurer.DockerCommandf("image inspect %s --format '{{.ID}}'", i)) == nil
}

// PullImages pulls multiple images parallelly by using a worker pool
func PullImages(h *api.Host, images []*Image) error {
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

// RetagAllToRepository retags all images in a list to another repository
func RetagAllToRepository(h *api.Host, images []*Image, repo string) error {
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

// AllToRepository generates a new list with a different repository
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
