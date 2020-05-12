package configurer

import (
	"github.com/Mirantis/mcc/pkg/host"
	log "github.com/sirupsen/logrus"
)

type UbuntuConfigurer struct {
	host *host.Host
}

func NewUbuntuConfigurer(host *host.Host) HostConfigurer {
	return &UbuntuConfigurer{
		host: host,
	}
}

func (u *UbuntuConfigurer) InstallEngine() error {
	err := u.host.Exec("sudo apt-get update && sudo apt-get install -y curl apt-utils")
	if err != nil {
		return err
	}

	err = u.host.Exec("curl https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh | DOCKER_URL=http://repos-internal.mirantis.com.s3.amazonaws.com CHANNEL=test bash")
	if err != nil {
		return err
	}

	err = u.host.Exec("sudo systemctl enable --now docker")
	if err != nil {
		return err
	}

	log.Infof("Succesfully installed engine on %s", u.host.Address)
	return nil
}
