package dummy

import (
	"github.com/Mirantis/mcc/pkg/product/dummy/api"
	"gopkg.in/yaml.v2"
)

// Dummy is the product
type Dummy struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// ClusterName returns the cluster name
func (p *Dummy) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// NewDummy returns a new instance of the dummy product
func NewDummy(data []byte) (*Dummy, error) {
	c := api.ClusterConfig{}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, err
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &Dummy{ClusterConfig: c}, nil
}

// Init returns an example configuration
func Init(kind string) *api.ClusterConfig {
	return api.Init(kind)
}
