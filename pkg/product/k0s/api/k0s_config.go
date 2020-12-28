package api

import "github.com/Mirantis/mcc/pkg/product/common/api"

// K0sConfig holds configuration for bootstraping k0s cluster
type K0sConfig struct {
	Version  string          `yaml:"version"`
	Config   api.GenericHash `yaml:"k0s"`
	Metadata K0sMetadata     `yaml:"-"`
}

// K0sMetadata information about k0s cluster install information
type K0sMetadata struct {
	Installed        bool
	InstalledVersion string
	ClusterID        string
	JoinToken        string
}
