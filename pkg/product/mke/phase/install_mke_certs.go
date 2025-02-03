package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// InstallMKE is the phase implementation for running the actual MKE installer container.
type InstallMKECerts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title.
func (p *InstallMKECerts) Title() string {
	return "Install MKE certificates"
}

// Run the installer container.
func (p *InstallMKECerts) ShouldRun() bool {
	if p.Config.Spec.MKE.CACertData == "" || p.Config.Spec.MKE.CertData == "" || p.Config.Spec.MKE.KeyData == "" {
		log.Debug("no MKE cert data to install")
		return false
	}
	return true
}

func (p *InstallMKECerts) Run() (err error) {
	log.Debug("adding install flag '--external-server-cert'. If this is an upgrade, then external certs must already be enabled.")
	p.Config.Spec.MKE.InstallFlags.AddUnlessExist("--external-server-cert")

	return p.installCertificates(p.Config)
}

// installCertificates installs user supplied MKE certificates.
func (p *InstallMKECerts) installCertificates(config *api.ClusterConfig) error {
	log.Infof("Installing MKE certificates")
	managers := config.Spec.Managers()
	err := managers.ParallelEach(func(h *api.Host) error {
		err := h.Exec(h.Configurer.DockerCommandf("volume inspect ucp-controller-server-certs"))
		if err != nil {
			log.Infof("%s: creating ucp-controller-server-certs volume", h)
			if err := h.Exec(h.Configurer.DockerCommandf("volume create ucp-controller-server-certs")); err != nil {
				return fmt.Errorf("create ucp-controller-server-certs volume: %w", err)
			}
		}

		dir, err := h.ExecOutput(h.Configurer.DockerCommandf(`volume inspect ucp-controller-server-certs --format "{{ .Mountpoint }}"`))
		if err != nil {
			return fmt.Errorf("get ucp-controller-server-certs volume mountpoint: %w", err)
		}

		log.Infof("%s: installing certificate files to %s", h, dir)
		if err := h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "ca.pem"), config.Spec.MKE.CACertData, "0600"); err != nil {
			return fmt.Errorf("write ca.pem: %w", err)
		}
		if err := h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "cert.pem"), config.Spec.MKE.CertData, "0600"); err != nil {
			return fmt.Errorf("write cert.pem: %w", err)
		}
		if err := h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "key.pem"), config.Spec.MKE.KeyData, "0600"); err != nil {
			return fmt.Errorf("write key.pem: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("install certificates: %w", err)
	}

	return nil
}
