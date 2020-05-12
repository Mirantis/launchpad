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

	output, err := u.host.ExecWithOutput("curl https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh | DOCKER_URL=http://repos-internal.mirantis.com.s3.amazonaws.com CHANNEL=test bash")
	if err != nil {
		log.Warnf("Failed to install engine: %s", err.Error())
		log.Warnln(output)
	}

	log.Infof("Succesfully installed engine on %s", u.host.Address)
	return nil
}
