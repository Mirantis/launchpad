package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// LabelNodes phase implementation
type LabelNodes struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *LabelNodes) Title() string {
	return "Label nodes"
}

// Run labels all nodes with launchpad label
func (p *LabelNodes) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	err := p.labelCurrentNodes(p.Config, swarmLeader)
	if err != nil {
		return err
	}

	return nil
}

func (p *LabelNodes) labelCurrentNodes(config *api.ClusterConfig, swarmLeader *api.Host) error {
	for _, h := range config.Spec.Hosts {
		nodeID, err := swarm.NodeID(h)
		if err != nil {
			return err
		}
		log.Infof("%s: labeling node", h)
		if h.Role == "msr" {
			// Add the MSR label in addition to the managed label
			msrLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedMSRLabelCmd, nodeID)
			err = swarmLeader.Exec(msrLabelCmd)
			if err != nil {
				return fmt.Errorf("Failed to label node %s as MSR (%s)", h, nodeID)
			}
		}
		labelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedLabelCmd, nodeID)
		err = swarmLeader.Exec(labelCmd)
		if err != nil {
			return fmt.Errorf("Failed to label node %s (%s)", h, nodeID)
		}
	}
	return nil
}
