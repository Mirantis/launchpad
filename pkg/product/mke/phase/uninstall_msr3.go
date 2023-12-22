package phase

import (
	"context"
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
	MSR3Phase

	leader *api.Host
}

// Title prints the phase title.
func (p *UninstallMSR3) Title() string {
	return "Uninstall MSR3 components"
}

func (p *UninstallMSR3) ShouldRun() bool {
	p.leader = p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR3() && p.leader.MSRMetadata.Installed
}

func (p *UninstallMSR3) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	p.leader = p.Config.Spec.MSRLeader()

	var err error

	p.kube, p.helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return err
	}

	return nil
}

// Run an uninstall by deleting the MSR CR referenced in the config.
func (p *UninstallMSR3) Run() error {
	ctx := context.Background()

	// Remove the LB service if it's being used, ignoring if it's not found.
	if p.Config.Spec.MSR.MSR3Config.ShouldConfigureLB() {
		err := p.kube.DeleteService(ctx, constant.ExposedLBServiceName)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	chartsToUninstall := p.Config.Spec.MSR.MSR3Config.Dependencies.List()

	// Add the storage provisioner chart to the list of charts to uninstall
	// if its configured.
	if p.Config.Spec.MSR.MSR3Config.ShouldConfigureStorageClass() {
		_, chartDetails := storageProvisionerChart(p.Config)
		chartsToUninstall = append(chartsToUninstall, chartDetails)
	}

	for _, d := range chartsToUninstall {
		// Uninstalling the msr-operator chart will remove the CRD which
		// will cause the MSR CR to be deleted.
		err := p.helm.Uninstall(ctx, &helm.Options{ChartDetails: *d})
		if err != nil {
			return fmt.Errorf("failed to uninstall Helm dependency: %q: %w", d.ReleaseName, err)
		}
	}

	return nil
}
