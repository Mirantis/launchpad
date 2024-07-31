package product

import "github.com/Mirantis/mcc/pkg/product/mke/api"

// Product is an interface that represents a product that launchpad can manage.
type Product interface {
	Apply(disableCleanup, force bool, concurrency int, forceUpgrade bool) error
	Reset() error
	Describe(reportName string) error
	ClientConfig() error
	Exec(target []string, interactive, first, all, parallel bool, role api.RoleType, os, cmd string) error
	ClusterName() string
}
