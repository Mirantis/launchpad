package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// LabelNodes phase implementation
type LabelNodes struct {
	Analytics
}

// Title for the phase
func (p *LabelNodes) Title() string {
	return "Label nodes"
}

// Run labels all nodes with launchpad label
func (p *LabelNodes) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	err := p.labelCurrentNodes(config, swarmLeader)
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
		labelCmd := swarmLeader.Configurer.DockerCommandf("node update --label-add com.mirantis.launchpad.managed=true %s", nodeID)
		err = swarmLeader.Exec(labelCmd)
		if err != nil {
			return fmt.Errorf("Failed to label node %s (%s)", h.Address, nodeID)
		}
	}
	return nil
}
