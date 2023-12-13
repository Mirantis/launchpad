package phase

import (
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

// ConfigureDeps phase implementation configures any Helm dependencies for
// msr-operator to be able to deploy and run an MSR CR.
type ConfigureDepsMSR3 struct {
	phase.Analytics
	phase.BasicPhase
	MSR3Phase

	leader *api.Host
}

// Title for the phase.
func (p *ConfigureDepsMSR3) Title() string {
	return "Configuring MSR3 dependencies"
}

// ShouldRun determines whether to attempt to configure dependencies by
// determining whether they have been installed already.
func (p *ConfigureDepsMSR3) ShouldRun() bool {
	if p.leader.MSRMetadata.Installed {
		return false
	}

	for _, chd := range p.Config.Spec.MSR.MSR3Config.Dependencies.List() {
		vers, err := version.NewSemver(chd.Version)
		if err != nil {
			log.Warnf("failed to parse version %q for dependency %q: %s", chd.Version, chd.ReleaseName, err)
			return true
		}

		needsUpgrade, err := p.Helm.ChartNeedsUpgrade(p.ctx, chd.ReleaseName, vers)
		if err != nil {
			log.Warnf("failed to check if dependency %q needs upgrade: %s", chd.ReleaseName, err)
			return true
		}

		if needsUpgrade {
			return true
		}
	}

	return false
}

// Run configures the dependencies for an MSR CR to be able to deploy by
// installing cert-manager, postgres-operator, rethinkdb-operator and
// msr-operator.  If these are already installed, the phase is a no-op.
func (p *ConfigureDepsMSR3) Run() error {
	for dep, chd := range p.Config.Spec.MSR.MSR3Config.Dependencies.List() {
		if _, err := p.Helm.Upgrade(p.ctx, &helm.Options{
			ChartDetails: chd,
			ReuseValues:  true,
			Wait:         true,
			Atomic:       true,
		}); err != nil {
			log.Infof("Dependency %q installed", dep)
		}
	}

	return nil
}
