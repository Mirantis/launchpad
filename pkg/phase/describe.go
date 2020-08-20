package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	cfg "github.com/Mirantis/mcc/pkg/config"
)

// Describe shows information about the current status of the cluster
type Describe struct{}

// Title for the phase
func (p *Describe) Title() string {
	return "Display cluster status"
}

// Run does the actual saving of the local state file
func (p *Describe) Run(config *api.ClusterConfig) error {
	urls := config.Spec.WebURLs()
	fmt.Println("Cluster:")
	fmt.Printf("* Name: %s\n", config.Metadata.Name)
	fmt.Printf("* Managers: %d\n", len(config.Spec.Managers()))
	fmt.Printf("* Workers: %d\n", len(config.Spec.Workers()))
	fmt.Printf("* DTR nodes: %d\n", len(config.Spec.Dtrs()))

	fmt.Println()
	fmt.Println("UCP:")
	if config.Spec.Ucp.Metadata.Installed {
		fmt.Printf("* Version: %s\n", config.Spec.Ucp.Metadata.InstalledVersion)
		fmt.Printf("* Admin UI: %s\n", urls.Ucp)
	} else {
		fmt.Println("* Version: not installed")
	}

	if cfg.ContainsDtr(*config) {
		fmt.Println()
		fmt.Println("DTR:")
		if config.Spec.Dtr.Metadata.Installed {
			fmt.Printf("* Version: %s\n", config.Spec.Dtr.Metadata.InstalledVersion)
			if urls.Dtr != "" {
				fmt.Printf("* Admin UI: %s\n", urls.Dtr)
			}
		} else {
			fmt.Println("* Version: not installed")
		}
	}

	for _, h := range config.Spec.Hosts {
		fmt.Println()
		fmt.Printf("Host '%s':\n", h.Address)
		fmt.Printf("* Role: %s\n", h.Role)
		fmt.Printf("* Hostname: %s\n", h.Metadata.LongHostname)
		fmt.Printf("* Internal address: %s\n", h.Metadata.InternalAddress)
		fmt.Printf("* OS: %s %s\n", h.Metadata.Os.ID, h.Metadata.Os.Version)
		version := h.Metadata.EngineVersion
		if version == "" {
			version = "not installed"
		}
		fmt.Printf("* Engine version: %s\n", version)
	}

	return nil
}
