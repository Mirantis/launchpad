package phase

import (
	"os"
	"path"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
)

// LoadImages phase uploads + docker loads images from host's imageDir to hosts.
type LoadImages struct {
	phase.Analytics
	phase.HostSelectPhase
}

// Title is the title for the phase.
func (p *LoadImages) Title() string {
	return "Upload images"
}

// HostFilterFunc returns true for hosts that have images to be uploaded.
func (p *LoadImages) HostFilterFunc(h *api.Host) bool {
	if h.ImageDir == "" {
		return false
	}
	log.Debugf("%s: listing images in imageDir '%s'", h, h.ImageDir)

	files, err := os.ReadDir(h.ImageDir)
	if err != nil {
		log.Errorf("%s: failed to list images in imageDir '%s': %s", h, h.ImageDir, err.Error())
		return false
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".tar" && ext != ".gz" {
			continue
		}

		imagePath := filepath.Join(h.ImageDir, entry.Name())
		h.Metadata.ImagesToUpload = append(h.Metadata.ImagesToUpload, imagePath)
		info, err := entry.Info()
		if err == nil {
			h.Metadata.TotalImageBytes += uint64(info.Size())
		}
	}

	return h.Metadata.TotalImageBytes > 0
}

// Prepare collects the hosts.
func (p *LoadImages) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Run does all the work.
func (p *LoadImages) Run() error {
	var totalBytes uint64
	p.Hosts.Each(func(h *api.Host) error {
		totalBytes += h.Metadata.TotalImageBytes
		return nil
	})

	log.Infof("total %s of images to upload", util.FormatBytes(totalBytes))

	return p.Hosts.Each(func(h *api.Host) error {
		for idx, f := range h.Metadata.ImagesToUpload {
			log.Debugf("%s: uploading image %d/%d", h, idx+1, len(h.Metadata.ImagesToUpload))

			base := path.Base(f)
			df := h.Configurer.JoinPath(h.Configurer.Pwd(h), base)
			err := h.WriteFileLarge(f, df)
			if err != nil {
				return err
			}

			log.Infof("%s: loading image %d/%d : %s", h, idx+1, len(h.Metadata.ImagesToUpload), base)
			err = h.Exec(h.Configurer.DockerCommandf("load -i %s", shellescape.Quote(base)))
			if err != nil {
				return err
			}
		}
		return nil
	})
}
