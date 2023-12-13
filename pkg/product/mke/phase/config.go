package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

type unexpectedConfigTypeError struct {
	Config interface{}
}

func (e *unexpectedConfigTypeError) Error() string {
	return fmt.Sprintf("unexpected config type: expected ClusterConfig, got %T", e.Config)
}

// ConvertConfigToClusterConfig performs a type assertion on a given config
// and returns an *api.ClusterConfig type.
func convertConfigToClusterConfig(config interface{}) (*api.ClusterConfig, error) {
	clusterConfig, ok := config.(*api.ClusterConfig)
	if !ok {
		return nil, &unexpectedConfigTypeError{config}
	}

	return clusterConfig, nil
}
