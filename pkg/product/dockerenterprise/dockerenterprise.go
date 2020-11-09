package dockerenterprise

import (
	"github.com/Mirantis/mcc/pkg/api"
	validator "github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

// DockerEnterprise is the product that bundles UCP, DTR and Docker Engine
type DockerEnterprise struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// ClusterName returns the cluster name
func (p *DockerEnterprise) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
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

// Init returns an example configuration
func Init() *api.ClusterConfig {
	return api.Init()
}
