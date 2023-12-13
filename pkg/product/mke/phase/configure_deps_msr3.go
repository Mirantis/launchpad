package phase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/mke"
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
}

// Title for the phase.
func (p *ConfigureDepsMSR3) Title() string {
	return "Configuring MSR3 dependencies"
}

func (p *ConfigureDepsMSR3) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)

	var err error

	p.kube, p.helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return err
	}

	return nil
}

// ShouldRun ...
func (p *ConfigureDepsMSR3) ShouldRun() bool {
	return true
}

// Run configures the dependencies for an MSR CR to be able to deploy by
// installing cert-manager, postgres-operator, rethinkdb-operator and
// msr-operator.  If these are already installed, the phase is a no-op.
func (p *ConfigureDepsMSR3) Run() error {
	for _, chd := range p.Config.Spec.MSR.MSR3Config.Dependencies.List() {
		vers, err := version.NewSemver(chd.Version)
		if err != nil {
			return fmt.Errorf("failed to parse version %q for dependency %q: %s", chd.Version, chd.ReleaseName, err)
		}

		needsUpgrade, err := p.helm.ChartNeedsUpgrade(context.Background(), chd.ReleaseName, vers)
		if err != nil {
			// Log any errors that are different then NotFound, but try to
			// upgrade anyway.
			var notFoundErr helm.ErrReleaseNotFound

			if !errors.As(err, &notFoundErr) {
				log.Debugf("failed to check if dependency %q needs upgrade, will try to upgrade anyway: %s", chd.ReleaseName, err)
			}

			needsUpgrade = true
		}

		if needsUpgrade {
			_, err := p.helm.Upgrade(context.Background(), &helm.Options{
				ChartDetails: *chd,
				ReuseValues:  true,
				Wait:         true,
				Atomic:       true,
			})
			if err != nil {
				return fmt.Errorf("failed to install/upgrade Helm release %q: %w", chd.ReleaseName, err)
			}

			log.Infof("dependency %q installed/upgraded", chd.ReleaseName)
		}
	}

	return nil
}
