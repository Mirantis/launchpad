package product

import (
	"github.com/Mirantis/mcc/pkg/config"
	de "github.com/Mirantis/mcc/pkg/product/docker_enterprise"
	"github.com/urfave/cli/v2"
)

// Product is an interface that represents a product that launchpad can manage.
type Product interface {
	Apply() error
	Reset() error
	Describe(reportName string) error
}

// GetProduct returns a product instance suitable for the configuration Kind
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
		return &de.DockerEnterprise{
			ClusterConfig: clusterConfig,
			SkipCleanup:   ctx.Bool("disable-cleanup"),
			Debug:         ctx.Bool("debug") || ctx.Bool("trace"),
		}, nil
	}
	return nil, nil

}
