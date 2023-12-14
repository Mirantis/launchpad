package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
)

// InstallOrUpgradeMSR3 deploys an MSR Custom Resource using the CRD provided
// within config.  It handles both the Install and Upgrade phases for MSR3.
type InstallOrUpgradeMSR3 struct {
	phase.Analytics
	phase.CleanupDisabling
	MSR3Phase

	leader *api.Host
}

// Title prints the phase title.
func (p *InstallOrUpgradeMSR3) Title() string {
	return "Configuring MSR3 Custom Resource"
}

// Prepare collects the hosts and labels them with the MSR role via the
// Kubernetes client so that they can be used as NodeSelector in the MSR CR.
func (p *InstallOrUpgradeMSR3) Prepare(config interface{}) error {
	p.leader = p.Config.Spec.MSRLeader()
	p.Config = config.(*api.ClusterConfig)

	var err error

	p.kube, p.helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return err
	}

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

		// If MKE is the target Kubernetes cluster, set the orchestrator
		// type to Kubernetes for the node.
		swarmLeader := p.Config.Spec.SwarmLeader()
		nodeID, err := swarm.NodeID(msrH)
		if err != nil {
			return fmt.Errorf("%s: failed to get node ID: %w", msrH, err)
		}

		kubeOrchestratorLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.KubernetesOrchestratorLabelCmd, nodeID)
		err = swarmLeader.Exec(kubeOrchestratorLabelCmd)
		if err != nil {
			return fmt.Errorf("failed to label node %s (%s) for kube orchestration: %w", hostname, nodeID, err)
		}
	}

	return nil
}

// ShouldRun should return true only if msr3 config is present.
func (p *InstallOrUpgradeMSR3) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR3()
}

// Run deploys an MSR CR to the cluster, setting NodeSelector to nodes the user
// has specified as MSR hosts in the config.
func (p *InstallOrUpgradeMSR3) Run() error {
	ctx := context.Background()

	h := p.leader

	if h.MSRMetadata == nil {
		h.MSRMetadata = &api.MSRMetadata{}
	}

	if err := p.Config.Spec.CheckMKEHealthRemote(h); err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", h)
	}

	if err := p.kube.ValidateMSROperatorReady(context.Background()); err != nil {
		return fmt.Errorf("failed to validate msr-operator is ready: %w", err)
	}

	msr := &p.Config.Spec.MSR.MSR3Config.MSR

	// Append the NodeSelector for the MSR hosts if not already present.
	if msr.Spec.NodeSelector == nil {
		msr.Spec.NodeSelector = make(map[string]string)
	}

	if _, ok := msr.Spec.NodeSelector[constant.MSRNodeSelector]; !ok {
		msr.Spec.NodeSelector[constant.MSRNodeSelector] = "true"
	}

	// Ensure the postgresql.spec.volume.size field is sane, postgres-operator
	// doesn't default the Size field and is picky about the format.
	if msr.Spec.Postgresql.Volume.Size == "" {
		msr.Spec.Postgresql.Volume.Size = "20Gi"
	}

	// Set the version tag to the desired MSR version specified in config.
	msr.Spec.Image.Tag = p.Config.Spec.MSR.Version

	// Configure Nginx.DNSNames if a LoadBalancerURL is specified.
	if p.Config.Spec.MSR.MSR3Config.ShouldConfigureLB() {
		msr.Spec.Nginx.DNSNames = append(msr.Spec.Nginx.DNSNames, p.Config.Spec.MSR.MSR3Config.LoadBalancerURL)
	}

	// TODO: Differentiate an upgrade from an install and set analytics
	// around that.
	if err := msr3.ApplyCRD(ctx, &p.Config.Spec.MSR.MSR3Config.MSR, p.kube); err != nil {
		return err
	}

	if p.Config.Spec.MSR.MSR3Config.ShouldConfigureLB() {
		if err := p.kube.ExposeLoadBalancer(ctx, p.Config.Spec.MSR.MSR3Config.LoadBalancerURL); err != nil {
			return fmt.Errorf("failed to expose msr via LoadBalancer: %w", err)
		}
	}

	msrMeta, err := msr3.CollectFacts(ctx, p.Config.Spec.MSR.MSR3Config.Name, p.kube, p.helm)
	if err != nil {
		return fmt.Errorf("failed to collect msr3 details: %w", err)
	}

	h.MSRMetadata = msrMeta

	return nil
}
