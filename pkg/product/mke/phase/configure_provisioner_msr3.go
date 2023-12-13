package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"
)

// ConfigureStorageProvisioner sets up the default provisioner to use based on
// the configured storage type.
type ConfigureStorageProvisioner struct {
	phase.Analytics
	phase.CleanupDisabling
	phase.KubernetesPhase

	leader *api.Host
	*storageProvisioner
}

func (p *ConfigureStorageProvisioner) Title() string {
	return "Configure Storage Provisioner"
}

func (p *ConfigureStorageProvisioner) Prepare(config interface{}) error {
	var err error

	p.Config, err = convertConfigToClusterConfig(config)
	if err != nil {
		return err
	}

	p.leader = p.Config.Spec.MSRLeader()

	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return fmt.Errorf("failed to get kube and helm clients: %w", err)
	}

	// Check to see if the storageProvisioner is already installed and at the
	// version we expect.  If the storageProvisioner is set to nil then we will
	// not run this phase.
	// TODO: We need to check for storageProvisioner changes as well during this
	// phase, potentially uninstalling a prior storageProvisioner if the type
	// has changed.  We can most likely add this when we support more than one
	// storageProvisioner type.
	p.storageProvisioner = storageProvisionerChart(p.Config)
	if p.storageProvisioner == nil {
		log.Debugf("no storage provisioner to configure")
		return nil
	}

	releases, err := p.Helm.List(fmt.Sprintf("^%s$", p.storageProvisioner.releaseDetails.ChartName))
	if err != nil {
		return fmt.Errorf("failed to list storage provisioner Helm releases: %w", err)
	}

	if len(releases) == 1 {
		if p.storageProvisioner.releaseDetails.Version != releases[0].Chart.Metadata.Version {
			log.Debugf("storage provisioner %q already installed, but at version %q, upgrading to %q", p.storageProvisioner.name, releases[0].Version, p.storageProvisioner.releaseDetails.Version)
			return nil
		}

		log.Debugf("storage provisioner %q already installed, at version: %s", p.storageProvisioner.name, p.storageProvisioner.releaseDetails.Version)
		p.storageProvisioner = nil
	}

	return nil
}

func (p *ConfigureStorageProvisioner) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR3() &&
		p.Config.Spec.MSR.V3.ShouldConfigureStorageClass() &&
		p.storageProvisioner != nil
}

func (p *ConfigureStorageProvisioner) Run() error {
	ctx := context.Background()

	if p.storageProvisioner != nil {
		if _, err := p.Helm.Upgrade(ctx, &helm.Options{
			ReleaseDetails: *p.storageProvisioner.releaseDetails,
			Timeout:        ptr.To(helm.DefaultTimeout),
		}); err != nil {
			return fmt.Errorf("failed to install storage provisioner %q: %w", p.storageProvisioner.name, err)
		}

		if err := p.Kube.SetStorageClassDefault(ctx, p.storageProvisioner.name); err != nil {
			return fmt.Errorf("failed to set default storage class to %q: %w", p.storageProvisioner.name, err)
		}
	}

	return nil
}

type storageProvisioner struct {
	name           string
	releaseDetails *helm.ReleaseDetails
}

// storageProvisionerChart returns a populated storageProvisioner type
// containing the name of the storage provisioner to use and the helm chart
// release details for that storage provisioner.  If the type if not supported,
// a warning is logged and a nil storageProvisioner is returned.
func storageProvisionerChart(config *api.ClusterConfig) *storageProvisioner {
	// TODO: Currently we only support "nfs" as a configured StorageClassType,
	// we should add some more.
	scType := config.Spec.MSR.V3.StorageClassType

	if scType == "" {
		log.Debugf("no storage class type configured, not configuring default storage class")
		return nil
	}

	log.Debugf("configuring default storage class for %q", scType)

	switch scType {
	case "nfs":
		return &storageProvisioner{
			name: "nfs-client",
			releaseDetails: &helm.ReleaseDetails{
				ChartName:   "nfs-subdir-external-provisioner",
				ReleaseName: "nfs-subdir-external-provisioner",
				RepoURL:     "https://kubernetes-sigs.github.io/nfs-subdir-external-provisioner/",
				Values: map[string]interface{}{
					"nfs": map[string]string{
						"server":     config.Spec.MSR.V3.StorageURL,
						"path":       "/",
						"volumeName": "nfs-subdir-external-provisioner-root",
					},
					"nodeSelector": map[string]string{"kubernetes.io/os": "linux"},
				},
				Version: "4.0.2",
			},
		}
	default:
		log.Warnf("unknown storage class type %q, not configuring default storage class", scType)
		return nil
	}
}
