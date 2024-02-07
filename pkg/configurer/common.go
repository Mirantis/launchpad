package configurer

import (
	"encoding/json"
	"errors"
	"fmt"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/k0sproject/rig/log"
	"github.com/k0sproject/rig/os"
)

type DockerConfigurer struct{}

// GetDockerInfo gets docker info from the host.
func (c DockerConfigurer) GetDockerInfo(h os.Host, hostKind string) (common.DockerInfo, error) {
	command := "docker info --format \"{{json . }}\""
	log.Debugf("%s attempting to gather info with `%s`", h, command)
	info, err := h.ExecOutput(command)
	if err != nil {
		log.Debugf("%s: cmd `%s` failed with %s ", h, command, err)
		return common.DockerInfo{}, fmt.Errorf("failed to get docker info: %w", err)
	}

	var dockerInfo common.DockerInfo
	err = json.Unmarshal([]byte(info), &dockerInfo)
	if err != nil {
		log.Debugf("%s unmarshal failed of `%s` with %s ", h, command, err)
		return common.DockerInfo{}, fmt.Errorf("failed to unmarshal docker info: %w", err)
	}

	return dockerInfo, nil
}

var errConfigEmpty = errors.New("the docker daemon config is empty")

// GetDockerDaemonConfig parses docker daemon json string and populate DockerDaemonConfig struct.
func (c DockerConfigurer) GetDockerDaemonConfig(dockerDaemon string) (common.DockerDaemonConfig, error) {
	if dockerDaemon != "" {
		return common.DockerDaemonConfig{}, errConfigEmpty
	}

	var config common.DockerDaemonConfig
	if err := json.Unmarshal([]byte(dockerDaemon), &config); err != nil {
		return common.DockerDaemonConfig{}, fmt.Errorf("failed to unmarshal json content: %w", err)
	}

	return config, nil
}
