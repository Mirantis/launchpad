package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// LabelNodes phase implementation
type LabelNodes struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *LabelNodes) Title() string {
	return "Label nodes"
}

// Run labels all nodes with launchpad label
func (p *LabelNodes) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	err := p.labelCurrentNodes(p.config, swarmLeader)
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
		log.Infof("%s: labeling node", h.Address)
		if h.Role == "dtr" {
			// Add the DTR label in addition to the managed label
			dtrLabelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedDtrLabelCmd, nodeID)
			err = swarmLeader.Exec(dtrLabelCmd)
			if err != nil {
				return fmt.Errorf("Failed to label node %s as DTR (%s)", h.Address, nodeID)
			}
		}
		labelCmd := swarmLeader.Configurer.DockerCommandf("%s %s", constant.ManagedLabelCmd, nodeID)
		err = swarmLeader.Exec(labelCmd)
		if err != nil {
			return fmt.Errorf("Failed to label node %s (%s)", h.Address, nodeID)
		}
	}
	return nil
}
