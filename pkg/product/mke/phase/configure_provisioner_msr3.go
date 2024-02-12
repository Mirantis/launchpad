package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
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
	ctx := context.Background()

	if _, ok := config.(*api.ClusterConfig); !ok {
		return fmt.Errorf("expected ClusterConfig, got %T", config)
	}

	p.Config = config.(*api.ClusterConfig)
	p.leader = p.Config.Spec.MSRLeader()

	var err error

	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return err
	}

	// Check to see if the storageProvisioner is already installed and at the
	// version we expect.  If the storageProvisioner is set to nil then we will
	// not run this phase.
	// TODO: We need to check for storageProvisioner changes as well during this
	// phase, potentially uninstalling a prior storageProvisioner if the type
	// has changed.  We can most likely add this when we support more than one
	// storageProvisioner type.
	sp := storageProvisionerChart(p.Config)
	if sp == nil {
		log.Debugf("no storage provisioner to configure")
		return nil
	}

	releases, err := p.Helm.List(ctx, fmt.Sprintf("^%s$", sp.releaseDetails.ChartName))
	if err != nil {
		return fmt.Errorf("failed to list storage provisioner Helm releases: %w", err)
	}

	if len(releases) > 1 {
		return fmt.Errorf("found more than one release for storage provisioner %q", sp.name)
	}

	if len(releases) == 1 {
		if sp.releaseDetails.Version != releases[0].Chart.Metadata.Version {
			log.Debugf("storage provisioner %q already installed, but at version %q, upgrading to %q", sp.name, releases[0].Version, sp.releaseDetails.Version)
			return nil
		}

		log.Debugf("storage provisioner %q already installed, at version: %s", sp.name, sp.releaseDetails.Version)
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
			return err
		}

		if err := p.Kube.SetStorageClassDefault(ctx, p.storageProvisioner.name); err != nil {
			return err
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
