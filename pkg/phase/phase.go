package phase

import "github.com/Mirantis/mcc/pkg/config"

// Phase interface
type Phase interface {
	Run(config *config.ClusterConfig) error
	Title() string
}
