package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/phase"
)

// UninstallMSR is the phase implementation for running MSR uninstall.
type UninstallMSR3 struct {
	phase.Analytics
	MSR3Phase
}

// Title prints the phase title.
func (p *UninstallMSR3) Title() string {
	return "Uninstall MSR3 components"
}

// Run an uninstall by deleting the MSR CR referenced in the config.
func (p *UninstallMSR3) Run() error {
	// Remove the MSR CR.
	n := p.Config.Spec.MSR.MSR3Config.MSR.Name

	if err := p.Kube.DeleteMSRCR(p.ctx, n); err != nil {
		return fmt.Errorf("failed to delete MSR CR: %q: %w", n, err)
	}

	// Remove Helm dependencies.
	for _, d := range p.Config.Spec.MSR.MSR3Config.Dependencies.List() {
		if err := p.Helm.Uninstall(p.ctx, &helm.Options{
			ChartDetails: d,
		}); err != nil {
			return fmt.Errorf("failed to delete Helm dependency: %q: %w", d.ReleaseName, err)
		}
	}

	return nil
}
