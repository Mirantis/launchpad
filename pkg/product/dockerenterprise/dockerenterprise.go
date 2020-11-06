package dockerenterprise

import (
	"fmt"
	"os"
	"path"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/constant"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/util"
	validator "github.com/go-playground/validator/v10"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// DockerEnterprise is the product that bundles UCP, DTR and Docker Engine
type DockerEnterprise struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// NewDockerEnterprise returns a new instance of the Docker Enterprise product
func NewDockerEnterprise(data []byte) (*DockerEnterprise, error) {
	c := api.ClusterConfig{}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	if err := validate(&c); err != nil {
		return nil, err
	}
	return &DockerEnterprise{ClusterConfig: c}, nil
}

// Validate validates that everything in the config makes sense
// Currently we do only very "static" validation using https://github.com/go-playground/validator
func validate(c *api.ClusterConfig) error {
	validator := validator.New()
	validator.RegisterStructValidation(requireManager, api.ClusterSpec{})
	return validator.Struct(c)
}

func requireManager(sl validator.StructLevel) {
	hosts := sl.Current().Interface().(api.ClusterSpec).Hosts
	if hosts.Count(func(h *api.Host) bool { return h.Role == "manager" }) == 0 {
		sl.ReportError(hosts, "Hosts", "", "manager required", "")
	}
}

const fileMode = 0700

func addFileLogger(clusterName string) (*os.File, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	clusterDir := path.Join(home, constant.StateBaseDir, "cluster", clusterName)
	if err := util.EnsureDir(clusterDir); err != nil {
		return nil, fmt.Errorf("error while creating directory for apply logs: %w", err)
	}
	logFileName := path.Join(clusterDir, "apply.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)

	if err != nil {
		return nil, fmt.Errorf("Failed to create apply log at %s: %s", logFileName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed
	log.AddHook(mcclog.NewFileHook(logFile))

	return logFile, nil
}

// Init returns an example configuration
func Init() *api.ClusterConfig {
	return api.Init()
}
