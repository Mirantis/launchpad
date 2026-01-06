package configurer

import (
	"encoding/json"
	"errors"
	"fmt"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/k0sproject/rig/log"
	"github.com/k0sproject/rig/os"
)

type DockerConfigurer struct{}

// GetDockerInfo gets docker info from the host.
func (c DockerConfigurer) GetDockerInfo(h os.Host) (commonconfig.DockerInfo, error) {
	command := "docker info --format \"{{json . }}\""
	log.Debugf("%s attempting to gather info with `%s`", h, command)
	info, err := h.ExecOutput(command)
	if err != nil {
		log.Debugf("%s: cmd `%s` failed with %s ", h, command, err)
		return commonconfig.DockerInfo{}, fmt.Errorf("failed to get docker info: %w", err)
	}

	var dockerInfo commonconfig.DockerInfo
	err = json.Unmarshal([]byte(info), &dockerInfo)
	if err != nil {
		log.Debugf("%s unmarshal failed of `%s` with %s ", h, command, err)
		return commonconfig.DockerInfo{}, fmt.Errorf("failed to unmarshal docker info: %w", err)
	}

	return dockerInfo, nil
}

var errConfigEmpty = errors.New("the docker daemon config is empty")

// GetDockerDaemonConfig parses docker daemon json string and populate DockerDaemonConfig struct.
func (c DockerConfigurer) GetDockerDaemonConfig(dockerDaemon string) (commonconfig.DockerDaemonConfig, error) {
	if dockerDaemon != "" {
		return commonconfig.DockerDaemonConfig{}, errConfigEmpty
	}

	var config commonconfig.DockerDaemonConfig
	if err := json.Unmarshal([]byte(dockerDaemon), &config); err != nil {
		return commonconfig.DockerDaemonConfig{}, fmt.Errorf("failed to unmarshal json content: %w", err)
	}

	return config, nil
}
