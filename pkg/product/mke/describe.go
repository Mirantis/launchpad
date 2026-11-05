package mke

import (
	"fmt"
	"os"

	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/phase"
	de "github.com/Mirantis/launchpad/pkg/product/mke/phase"
	"github.com/Mirantis/launchpad/pkg/util/redact"
	rigcmd "github.com/k0sproject/rig/v2/cmd"
	"gopkg.in/yaml.v2"
)

// Describe - gets information about configured instance.
func (p *MKE) Describe(reportName string) error {
	var msr bool
	var mke bool

	if reportName == "msr" {
		msr = true
	}

	if reportName == "mke" {
		mke = true
	}

	if reportName == "config" {
		encoded, err := yaml.Marshal(p.ClusterConfig)
		if err != nil {
			return fmt.Errorf("failed to encode cluster config: %w", err)
		}

		// --disable-redact is the documented opt-out for hiding sensitive
		// information in launchpad's output; the config debug dump honours
		// it the same way.
		if !rigcmd.DisableRedact {
			encoded, err = redact.YAML(encoded)
			if err != nil {
				return fmt.Errorf("failed to redact cluster config: %w", err)
			}
		}

		if _, err := os.Stdout.Write(encoded); err != nil {
			return fmt.Errorf("failed to write cluster config: %w", err)
		}

		return nil
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&de.DetectOS{},
		&de.GatherFacts{},
		&common.Disconnect{},
		&de.Describe{MKE: mke, MSR: msr},
	)

	if err := phaseManager.Run(); err != nil {
		return fmt.Errorf("failed to describe cluster: %w", err)
	}
	return nil
}
