package product

// Product is an interface that represents a product that launchpad can manage.
type Product interface {
	Apply(disableCleanup, force bool) error
	Reset() error
	Describe(reportName string) error
	ClientConfig() error
	Exec(target []string, interactive, first, all, parallel bool, role, os, cmd string) error
	ClusterName() string
}
