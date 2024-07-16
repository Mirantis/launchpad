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
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	return "Configuring MSR Custom Resource"
}

// Prepare collects the hosts and labels them with the MSR role via the
// Kubernetes client so that they can be used as NodeSelector in the MSR CR.
func (p *InstallOrUpgradeMSR3) Prepare(config interface{}) error {
	var err error

	p.Config, err = convertConfigToClusterConfig(config)
	if err != nil {
		return err
	}

	msr3s := p.Config.Spec.MSR3s()
	p.leader = msr3s.First()

	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return fmt.Errorf("failed to get kube and helm clients: %w", err)
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

	if h.MSR3Metadata == nil {
		h.MSR3Metadata = &api.MSR3Metadata{}
	}

	var msr3Hosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == api.RoleMSR3 {
			msr3Hosts = append(msr3Hosts, h)
		}
	}

	for _, msrH := range msr3Hosts {
		hostname := msrH.Metadata.Hostname

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

		err = p.Kube.PrepareNodeForMSR(context.Background(), hostname)
		if err != nil {
			return fmt.Errorf("%s: failed to label node: %w", msrH, err)
		}
	}

	if err := p.Config.Spec.CheckMKEHealthRemote([]*api.Host{h}); err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installation flag and check connectivity: %w", h, err)
	}

	if err := p.Kube.ValidateMSROperatorReady(ctx); err != nil {
		return fmt.Errorf("failed to validate msr-operator is ready: %w", err)
	}

	msr := p.Config.Spec.MSR3.CRD

	// Append the NodeSelector for the MSR hosts if not already present.
	nodeSelector, found, err := unstructured.NestedMap(msr.Object, "spec", "nodeSelector")
	if err != nil {
		return fmt.Errorf("failed to get MSR spec.nodeSelector: %w", err)
	}

	if !found || nodeSelector == nil {
		nodeSelector = make(map[string]interface{})
	}

	if _, ok := nodeSelector[constant.MSRNodeSelector]; !ok {
		nodeSelector[constant.MSRNodeSelector] = "true"
	}

	// Ensure the postgresql.spec.volume.size field is sane, postgres-operator
	// doesn't default the Size field and is picky about the format.
	postgresVolumeSize, found, err := unstructured.NestedString(msr.Object, "spec", "postgresql", "volume", "size")
	if err != nil {
		return fmt.Errorf("failed to get MSR spec.postgresql.volume.size: %w", err)
	}

	if !found || postgresVolumeSize == "" {
		if err := unstructured.SetNestedField(msr.Object, "20Gi", "spec", "postgresql", "volume", "size"); err != nil {
			return fmt.Errorf("failed to set MSR spec.postgresql.volume.size: %w", err)
		}
	}

	// Set the version tag to the desired MSR version specified in config.
	if err := unstructured.SetNestedField(msr.Object, p.Config.Spec.MSR3.Version, "spec", "image", "tag"); err != nil {
		return fmt.Errorf("failed to set MSR spec.image.tag: %w", err)
	}

	// Configure Nginx.DNSNames if a LoadBalancerURL is specified.
	if p.Config.Spec.MSR3.ShouldConfigureLB() {
		if err := unstructured.SetNestedStringSlice(msr.Object, []string{"nginx", "localhost", p.Config.Spec.MSR3.LoadBalancerURL}, "spec", "nginx", "dnsNames"); err != nil {
			return fmt.Errorf("failed to set MSR spec.nginx.dnsNames to include LoadBalancerURL: %q: %w", p.Config.Spec.MSR3.LoadBalancerURL, err)
		}
	}

	// TODO: Differentiate an upgrade from an install and set analytics
	// around that.
	if err := msr3.ApplyCRD(ctx, msr, p.Kube); err != nil {
		return fmt.Errorf("failed to apply MSR CRD: %w", err)
	}

	if p.Config.Spec.MSR3.ShouldConfigureLB() {
		if err := p.Kube.ExposeLoadBalancer(ctx, p.Config.Spec.MSR3.LoadBalancerURL); err != nil {
			log.Warnf("failed to expose MSR via LoadBalancer: %s", err)
		}
	}

	rc, err := p.Kube.GetMSRResourceClient()
	if err != nil {
		return fmt.Errorf("failed to get MSR resource client: %w", err)
	}

	msrMeta, err := msr3.CollectFacts(ctx, p.Config.Spec.MSR3.CRD.GetName(), p.Kube, rc, p.Helm)
	if err != nil {
		return fmt.Errorf("failed to collect MSR details: %w", err)
	}

	h.MSR3Metadata = msrMeta

	return nil
}
