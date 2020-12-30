package mke

import (
	"bytes"
	"io"

	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"gopkg.in/yaml.v2"
)

// MKE is the product
type MKE struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// ClusterName returns the cluster name
func (p *MKE) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// New returns a new instance of the Docker Enterprise product
func New(data []byte) (*MKE, error) {
	c := api.ClusterConfig{}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.SetStrict(true)
	if err := dec.Decode(&c); err != nil && err != io.EOF {
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
