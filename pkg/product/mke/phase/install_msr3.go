package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// InstallMSR3 deploys an MSR Custom Resource using the CRD provided within
// config.
type InstallMSR3 struct {
	phase.Analytics
	phase.CleanupDisabling
	MSR3Phase

	leader *api.Host
}

// Title prints the phase title.
func (p *InstallMSR3) Title() string {
	return "Deploy MSR3 Custom Resource"
}

// Prepare collects the hosts and labels them with the MSR role via the
// Kubernetes client so that they can be used as NodeSelector in the MSR CR.
func (p *InstallMSR3) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)

	var msrHosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == "msr" {
			msrHosts = append(msrHosts, h)
		}
	}

	for _, msrH := range msrHosts {
		hostname := msrH.Metadata.Hostname

		err := p.kube.LabelNode(context.Background(), hostname)
		if err != nil {
			return fmt.Errorf("%s: failed to label node: %s", msrH, err.Error())
		}
	}

	return nil
}

// ShouldRun should return true only when there is an installation to be
// performed.  An installation is required when the MSR CR is not present.
func (p *InstallMSR3) ShouldRun() bool {
	p.leader = p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR() && (p.leader.MSRMetadata == nil || !p.leader.MSRMetadata.Installed)
}

// Run deploys an MSR CR to the cluster, setting NodeSelector to nodes the user
// has specified as MSR hosts in the config.
func (p *InstallMSR3) Run() error {
	ctx := context.Background()

	h := p.leader

	if h.MSRMetadata == nil {
		h.MSRMetadata = &api.MSRMetadata{}
	}

	if err := p.Config.Spec.CheckMKEHealthRemote(h); err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", h)
	}

	if err := p.ApplyCRD(ctx); err != nil {
		return err
	}

	msrMeta, err := msr3.CollectFacts(context.Background(), p.Config.Spec.MSR.MSR3Config.Name, p.kube, p.helm)
	if err != nil {
		return fmt.Errorf("failed to collect msr3 details: %w", err)
	}

	h.MSRMetadata = msrMeta

	return nil
}
