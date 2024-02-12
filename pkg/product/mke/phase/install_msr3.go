package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

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
	phase.KubernetesPhase

	leader *api.Host
}

// Title prints the phase title.
func (p *InstallOrUpgradeMSR3) Title() string {
	return "Configure MSR Custom Resource"
}

// Prepare collects the hosts and labels them with the MSR role via the
// Kubernetes client so that they can be used as NodeSelector in the MSR CR.
func (p *InstallOrUpgradeMSR3) Prepare(config interface{}) error {
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

// ShouldRun should return true only if MSR3 config is present.
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

	var msrHosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == "msr" {
			msrHosts = append(msrHosts, h)
		}
	}

	for _, msrH := range msrHosts {
		hostname := msrH.Metadata.Hostname

		swarmLeader := p.Config.Spec.SwarmLeader()
		nodeID, err := swarm.NodeID(msrH)
		if err != nil {
			return fmt.Errorf("%s: failed to get node ID: %w", msrH, err)
		}

		// Set the orchestrator type to Kubernetes for the node.
		kubeOrchestratorLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.KubernetesOrchestratorLabelCmd, nodeID)
		err = swarmLeader.Exec(kubeOrchestratorLabelCmd)
		if err != nil {
			return fmt.Errorf("failed to label node %s (%s) for kube orchestration: %w", hostname, nodeID, err)
		}

		// Remove Swarm as an orchestrator type for the node (if present).
		swarmOrchestratorCheckLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.SwarmOrchestratorCheckLabelCmd, nodeID)
		output, err := swarmLeader.ExecOutput(swarmOrchestratorCheckLabelCmd)
		if err != nil {
			log.Warnf("failed to check swarm orchestrator label on node %s (%s): %s", hostname, nodeID, err)
		}

		if output != "" {
			swarmOrchestratorRemoveLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.SwarmOrchestratorRemoveLabelCmd, nodeID)
			err = swarmLeader.Exec(swarmOrchestratorRemoveLabelCmd)
			if err != nil {
				log.Warnf("failed to remove swarm orchestrator label from node %s (%s): %s", hostname, nodeID, err)
			}
		}

		err = p.Kube.PrepareNodeForMSR(ctx, hostname)
		if err != nil {
			return fmt.Errorf("%s: failed to prepare node for MSR: %w", msrH, err)
		}

	}

	if err := p.Config.Spec.CheckMKEHealthRemote(h); err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installation flag and check connectivity: %w", h, err)
	}

	if err := p.Kube.ValidateMSROperatorReady(ctx); err != nil {
		return fmt.Errorf("failed to validate msr-operator is ready: %w", err)
	}

	msr := &p.Config.Spec.MSR.V3.CRD

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
		msr.Spec.Postgresql.Volume.Size = constant.DefaultPostgresVolumeSize
	}

	// Set the version tag to the desired MSR version specified in config.
	msr.Spec.Image.Tag = p.Config.Spec.MSR.Version

	// Configure Nginx.DNSNames if a LoadBalancerURL is specified.
	if p.Config.Spec.MSR.V3.ShouldConfigureLB() {
		msr.Spec.Nginx.DNSNames = append(msr.Spec.Nginx.DNSNames, p.Config.Spec.MSR.V3.LoadBalancerURL)
	}

	// TODO: Differentiate an upgrade from an install and set analytics
	// around that.
	if err := msr3.ApplyCRD(ctx, &p.Config.Spec.MSR.V3.CRD, p.Kube); err != nil {
		return err
	}

	if p.Config.Spec.MSR.V3.ShouldConfigureLB() {
		if err := p.Kube.ExposeLoadBalancer(ctx, p.Config.Spec.MSR.V3.LoadBalancerURL); err != nil {
			return fmt.Errorf("failed to expose MSR via LoadBalancer: %w", err)
		}
	}

	rc, err := p.Kube.GetMSRResourceClient()
	if err != nil {
		return fmt.Errorf("failed to get MSR resource client: %w", err)
	}

	msrMeta, err := msr3.CollectFacts(ctx, p.Config.Spec.MSR.V3.CRD.GetName(), p.Kube, rc, p.Helm)
	if err != nil {
		return fmt.Errorf("failed to collect MSR details: %w", err)
	}

	h.MSRMetadata = msrMeta

	return nil
}
