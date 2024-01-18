package docker

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/k0sproject/rig/os"
)

// DockerInfo structural data for information on a hosts docker implementation
type DockerInfo struct {
	DockerRootDir string `json:"DockerRootDir"`
}

// Info retrieve docker info data for a host
//
//	@NOTE this function uses rig.os.Host instead of product.mke.api.Host
func Info(h os.Host) (DockerInfo, error) {
	di := DockerInfo{}

	log.Infof("%s: Retrieving docker info", h)

	if output, err := h.ExecOutput("docker info --format=json"); err != nil {
		return di, fmt.Errorf("%s: failed to retrieve docker info: %s", h, err.Error())
	} else if err := json.Unmarshal([]byte(output), &di); err != nil {
		return di, fmt.Errorf("%s: failed to unmarshall docker info: %s", h, err.Error())
	}
	return di, nil
}
