package phase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// UninstallMSR is the phase implementation for running MSR uninstall.
type UninstallMSR3 struct {
	phase.Analytics
	phase.KubernetesPhase
}

// Title prints the phase title.
func (p *UninstallMSR3) Title() string {
	return "Uninstall MSR components"
}

func (p *UninstallMSR3) ShouldRun() bool {
	leader := p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR3() && leader.MSRMetadata.Installed
}

func (p *UninstallMSR3) Prepare(config interface{}) error {
	if _, ok := config.(*api.ClusterConfig); !ok {
		return errInvalidConfig
	}

	var err error

	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return fmt.Errorf("failed to get kube and helm clients: %w", err)
	}

	return nil
}

// Run an uninstall by deleting the MSR CR referenced in the config.
func (p *UninstallMSR3) Run() error {
	ctx := context.Background()

	var errs error

	// Remove the LB service if it's being used, ignoring if it's not found.
	if p.Config.Spec.MSR.V3.ShouldConfigureLB() {
		err := p.Kube.DeleteService(ctx, constant.ExposedLBServiceName)
		if err != nil && !apierrors.IsNotFound(err) {
			errs = errors.Join(errs)
		}
	}

	chartsToUninstall := p.Config.Spec.MSR.V3.Dependencies.List()

	// Add the storage provisioner chart to the list of charts to uninstall
	// if it's configured.
	if p.Config.Spec.MSR.V3.ShouldConfigureStorageClass() {
		sp := storageProvisionerChart(p.Config)

		if sp != nil {
			chartsToUninstall = append(chartsToUninstall, sp.releaseDetails)
		}
	}

	for _, d := range chartsToUninstall {
		// Uninstalling the msr-operator chart will remove the CRD which
		// will cause the MSR CR to be deleted.
		err := p.Helm.Uninstall(&helm.Options{ReleaseDetails: *d})
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("%q: %w", d.ReleaseName, err))
		}
	}

	if errs != nil {
		return fmt.Errorf("failed to uninstall Helm dependencies: %w", errs)
	}

	return nil
}
