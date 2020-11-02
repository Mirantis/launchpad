package component

import (
	"github.com/Mirantis/mcc/pkg/component/ucp"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/urfave/cli/v2"
)

// Component an interface that represents all components that LP could be handling.
type Component interface {
	Apply() error
	Reset() error
	Describe(reportName string) error
}

// Components is an intializer for a component
func Components(ctx *cli.Context) (Component, error) {
	cfgData, err := config.ResolveClusterFile(ctx.String("config"))
	if err != nil {
		return nil, err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return nil, err
	}

	if err = config.Validate(&clusterConfig); err != nil {
		return nil, err
	}

	clusterConfig.Spec.Metadata.Force = ctx.Bool("force")

	//
	switch clusterConfig.Kind {
	case "DockerEnterprise":
		return ucp.UCP{
			ClusterConfig: clusterConfig,
			SkipCleanup:   ctx.Bool("disable-cleanup"),
			Debug:         ctx.Bool("debug") || ctx.Bool("trace"),
		}, nil
	}
	return nil, nil

}
