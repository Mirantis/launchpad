package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/utils/pointer"

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
}

func (p *ConfigureStorageProvisioner) Title() string {
	return "Configure Storage Provisioner"
}

func (p *ConfigureStorageProvisioner) Prepare(config interface{}) error {
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

	return nil
}

func (p *ConfigureStorageProvisioner) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR3() &&
		(p.leader.MSRMetadata == nil || !p.leader.MSRMetadata.Installed) &&
		p.Config.Spec.MSR.V3.ShouldConfigureStorageClass()
}

func (p *ConfigureStorageProvisioner) Run() error {
	ctx := context.Background()

	name, releaseDetails := storageProvisionerChart(p.Config)
	if releaseDetails == nil {
		log.Debugf("no storage provisioner to configure")
		return nil
	}

	if _, err := p.Helm.Upgrade(ctx, &helm.Options{
		ReleaseDetails: *releaseDetails,
		Timeout:        pointer.Duration(helm.DefaultTimeout),
	}); err != nil {
		return err
	}

	if err := p.Kube.SetStorageClassDefault(ctx, name); err != nil {
		return err
	}

	return nil
}

// storageProvisionerChart returns the name of the StorageClass and the
// helm chart details for the storage provisioner to configure based on the
// configured StorageClassType, if the type if not supported, a warning if
// logged and no chart is returned.
func storageProvisionerChart(config *api.ClusterConfig) (string, *helm.ReleaseDetails) {
	// TODO: Currently we only support "nfs" as a configured StorageClassType,
	// we should add some more.
	scType := config.Spec.MSR.V3.StorageClassType

	if scType == "" {
		log.Debugf("no storage class type configured, not configuring default storage class")
		return "", nil
	}

	log.Debugf("configuring default storage class for %q", scType)

	switch scType {
	case "nfs":
		return "nfs-client", &helm.ReleaseDetails{
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
		}
	default:
		log.Warnf("unknown storage class type %q, not configuring default storage class", scType)
		return "", nil
	}
}
