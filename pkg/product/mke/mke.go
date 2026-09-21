package mke

import (
	"fmt"
	"time"

	"github.com/Mirantis/launchpad/pkg/product/mke/config"
	"gopkg.in/yaml.v2"
)

// MKE is the product.
type MKE struct {
	ClusterConfig config.ClusterConfig
	// Timeout bounds the total wall-clock time Apply or Reset may spend
	// across all phases. Zero (the default) means no deadline. Set via
	// SetTimeout.
	Timeout time.Duration
}

// SetTimeout sets the overall deadline for Apply and Reset.
func (p *MKE) SetTimeout(timeout time.Duration) {
	p.Timeout = timeout
}

// ClusterName returns the cluster name.
func (p *MKE) ClusterName() string {
	return p.ClusterConfig.Metadata.Name
}

// NewMKE returns a new instance of the Docker Enterprise product.
func NewMKE(data []byte) (*MKE, error) {
	c := config.ClusterConfig{}
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse cluster config: %w", err)
	}

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate cluster config: %w", err)
	}
	return &MKE{ClusterConfig: c}, nil
}

// Init returns an example configuration.
func Init(kind string) *config.ClusterConfig {
	return config.Init(kind)
}
