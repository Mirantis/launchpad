package configurer

import (
	"encoding/json"
	"fmt"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/k0sproject/rig/log"
	"github.com/k0sproject/rig/os"
)

type DockerConfigurer struct{}

// GetDockerInfo gets docker info from the host
func (c DockerConfigurer) GetDockerInfo(h os.Host, hostKind string) (common.DockerInfo, error) {
	command := "docker info --format \"{{json . }}\""
	log.Infof("%s attempting to execute `docker info`", h)
	info, err := h.ExecOutput(command)

	if err != nil && hostKind != "windows" {
		log.Infof("%s attempting to execute `docker info` with sudo", h)
		info, err = h.ExecOutput(fmt.Sprintf("sudo %s", command))
		if err != nil {
			return common.DockerInfo{}, err
		}
	}

	var dockerInfo common.DockerInfo
	err = json.Unmarshal([]byte(info), &dockerInfo)
	if err != nil {
		log.Infof("%s no `docker info` found", h)
		return common.DockerInfo{}, err
	}

	return dockerInfo, nil
}

// GetDockerDaemonConfig parses docker daemon json string and populate DockerDaemonConfig struct
func (c DockerConfigurer) GetDockerDaemonConfig(dockerDaemon string) (common.DockerDaemonConfig, error) {
	if dockerDaemon != "" {
		return common.DockerDaemonConfig{}, fmt.Errorf("the docker daemon config is empty")
	}

	var config common.DockerDaemonConfig
	if err := json.Unmarshal([]byte(dockerDaemon), &config); err != nil {
		return common.DockerDaemonConfig{}, fmt.Errorf("failed to unmarshal json content: %w", err)
	}

	return config, nil
}
