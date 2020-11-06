package product

// Product is an interface that represents a product that launchpad can manage.
type Product interface {
	Apply() error
	Reset() error
	Describe(reportName string) error
	ClientConfig() error
	Exec(target string, interactive, first bool, role, cmd string) error
}
