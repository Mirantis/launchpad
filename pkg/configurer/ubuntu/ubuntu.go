package ubuntu

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/host"
	log "github.com/sirupsen/logrus"
)

type UbuntuConfigurer struct {
	configurer.LinuxConfigurer
}

func (u *UbuntuConfigurer) InstallEngine() error {
	err := u.Host.Exec("sudo apt-get update && sudo apt-get install -y curl apt-utils")
	if err != nil {
		return err
	}

	err = u.Host.Exec("curl https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh | DOCKER_URL=http://repos-internal.mirantis.com.s3.amazonaws.com CHANNEL=test bash")
	if err != nil {
		return err
	}

	err = u.Host.Exec("sudo systemctl enable --now docker")
	if err != nil {
		return err
	}

	log.Infof("Succesfully installed engine on %s", u.Host.Address)
	return nil
}

func resolveUbuntuConfigurer(h *host.Host) host.HostConfigurer {
	if h.Metadata.Os.ID != "ubuntu" {
		return nil
	}
	switch h.Metadata.Os.Version {
	case "18.04":
		configurer := &BionicConfigurer{
			UbuntuConfigurer: UbuntuConfigurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
		return configurer
	}

	return nil
}

func init() {
	host.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
