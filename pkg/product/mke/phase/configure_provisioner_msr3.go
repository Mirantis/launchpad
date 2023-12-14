package phase

import (
	"context"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// ConfigureStorageProvisioner sets up the default provisioner to use based on
// the configured storage type.
type ConfigureStorageProvisioner struct {
	phase.Analytics
	phase.CleanupDisabling
	MSR3Phase

	leader *api.Host
}

func (p *ConfigureStorageProvisioner) Title() string {
	return "Configure Storage Provisioner"
}

func (p *ConfigureStorageProvisioner) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)

	var err error

	p.kube, p.helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return err
	}

	return nil
}

func (p *ConfigureStorageProvisioner) ShouldRun() bool {
	p.leader = p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR3() &&
		(p.leader.MSRMetadata == nil || !p.leader.MSRMetadata.Installed) &&
		p.Config.Spec.MSR.MSR3Config.ShouldConfigureStorageClass()
}

func (p *ConfigureStorageProvisioner) Run() error {
	ctx := context.Background()

	// TODO: Currently we only support "nfs" as a configured StorageClassType,
	// we should add some more.
	if p.Config.Spec.MSR.MSR3Config.StorageClassType == "nfs" {
		log.Debug("configuring default storage class for NFS")

		p.helm.Upgrade(ctx, &helm.Options{
			ChartDetails: helm.ChartDetails{
				ChartName:   "nfs-subdir-external-provisioner",
				ReleaseName: "nfs-subdir-external-provisioner",
				RepoURL:     "https://kubernetes-sigs.github.io/nfs-subdir-external-provisioner/",
				Values: map[string]interface{}{
					"nfs.server":   p.Config.Spec.MSR.MSR3Config.StorageURL,
					"nfs.path":     "/",
					"nodeSelector": map[string]string{"kubernetes.io/os": "linux"},
				},
			},
		})

		if err := p.kube.SetStorageClassDefault(context.Background(), "nfs-client"); err != nil {
			return err
		}
	}

	return nil
}
