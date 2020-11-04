package product

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/product/ucp"
	"github.com/urfave/cli/v2"
)

// Product an interface that represents all products that LP could be handling.
type Product interface {
	Apply() error
	Reset() error
	Describe(reportName string) error
}

// GetProduct is an intializer for a product
func GetProduct(ctx *cli.Context) (Product, error) {
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

	switch clusterConfig.Kind {
	case "DockerEnterprise":
		return &ucp.UCP{
			ClusterConfig: clusterConfig,
			SkipCleanup:   ctx.Bool("disable-cleanup"),
			Debug:         ctx.Bool("debug") || ctx.Bool("trace"),
		}, nil
	}
	return nil, nil

}
