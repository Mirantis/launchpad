package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
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
func (p *InstallMKECerts) Run() (err error) {

	if p.Config.Spec.MKE.CACertData == "" || p.Config.Spec.MKE.CertData == "" || p.Config.Spec.MKE.KeyData == "" {
		log.Debug("Skipping cert install as some data is not available in config")
		return nil
	}

	log.Debug("Certificate installation data has been provided, so an install flag '--external-server-cert' will be added if this is a new installation")
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
			err := h.Exec(h.Configurer.DockerCommandf("volume create ucp-controller-server-certs"))
			if err != nil {
				return err
			}
		}

		dir, err := h.ExecOutput(h.Configurer.DockerCommandf(`volume inspect ucp-controller-server-certs --format "{{ .Mountpoint }}"`))
		if err != nil {
			return err
		}

		log.Infof("%s: installing certificate files to %s", h, dir)
		err = h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "ca.pem"), config.Spec.MKE.CACertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "cert.pem"), config.Spec.MKE.CertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(h, h.Configurer.JoinPath(dir, "key.pem"), config.Spec.MKE.KeyData, "0600")
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
