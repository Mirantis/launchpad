package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/Mirantis/launchpad/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// LabelNodes phase implementation.
type LabelNodes struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *LabelNodes) Title() string {
	return "Label nodes"
}

// Run labels all nodes with launchpad label.
func (p *LabelNodes) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	err := p.labelCurrentNodes(p.Config, swarmLeader)
	if err != nil {
		return err
	}

	return nil
}

func (p *LabelNodes) labelCurrentNodes(config *api.ClusterConfig, swarmLeader *api.Host) error {
	var sans []string
	for _, flag := range p.Config.Spec.MKE.InstallFlags {
		if strings.HasPrefix(flag, "--san") {
			sans = append(sans, common.FlagValue(flag))
		}
	}
	sanList := strings.Join(sans, ",")

	for _, h := range config.Spec.Hosts {
		nodeID, err := swarm.NodeID(h)
		if err != nil {
			return fmt.Errorf("failed to get node ID for %s: %w", h, err)
		}
		log.Infof("%s: labeling node", h)
		if h.Role == "manager" && len(sanList) > 0 {
			sanLabelCmd := swarmLeader.Configurer.DockerCommandf("node update --label-add com.docker.ucp.SANs=%s %s", sanList, nodeID)
			err = swarmLeader.Exec(sanLabelCmd)
			if err != nil {
				return fmt.Errorf("failed to add SANs label for node %s: %w", h, err)
			}
		}
		if h.Role == "msr" {
			// Add the MSR label in addition to the managed label
			msrLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedMSRLabelCmd, nodeID)
			err = swarmLeader.Exec(msrLabelCmd)
			if err != nil {
				return fmt.Errorf("failed to label node %s as MSR (%s): %w", h, nodeID, err)
			}
		}
		labelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedLabelCmd, nodeID)
		err = swarmLeader.Exec(labelCmd)
		if err != nil {
			return fmt.Errorf("failed to label node %s (%s): %w", h, nodeID, err)
		}
	}
	return nil
}
