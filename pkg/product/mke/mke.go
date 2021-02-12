package mke

import (
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"gopkg.in/yaml.v2"
)

// MKE is the product
type MKE struct {
	ClusterConfig api.ClusterConfig
}

// ClusterName returns the cluster name
func (p *MKE) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// NewMKE returns a new instance of the Docker Enterprise product
func NewMKE(data []byte) (*MKE, error) {
	c := api.ClusterConfig{}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, err
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &MKE{ClusterConfig: c}, nil
}

// Init returns an example configuration
func Init(kind string) *api.ClusterConfig {
	return api.Init(kind)
}
