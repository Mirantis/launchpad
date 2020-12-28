package k0s

import (
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	"gopkg.in/yaml.v2"
)

// K0s is the product
type K0s struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// ClusterName returns the cluster name
func (p *K0s) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// New returns a new instance of the Docker Enterprise product
func New(data []byte) (*K0s, error) {
	c := api.ClusterConfig{}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, err
	}

	return &K0s{ClusterConfig: c}, nil
}

// Init returns an example configuration
func Init(kind string) *api.ClusterConfig {
	return api.Init(kind)
}

// ClientConfig ...
func (p *K0s) ClientConfig() error {
	return nil
}
