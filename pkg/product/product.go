package product

import "time"

// Product is an interface that represents a product that launchpad can manage.
type Product interface {
	Apply(disableCleanup, force bool, concurrency int, forceUpgrade bool) error
	Reset() error
	Describe(reportName string) error
	ClientConfig() error
	Exec(target []string, interactive, first, all, parallel bool, role, os, cmd string) error
	ClusterName() string
	// SetTimeout bounds the total wall-clock time Apply or Reset may spend
	// across all phases. Zero (the default) means no deadline.
	SetTimeout(timeout time.Duration)
}
