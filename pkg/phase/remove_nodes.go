package phase

import (
	"fmt"
	"strings"
	"time"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RemoveNodes phase implementation
type RemoveNodes struct {
	Analytics
}

// Title for the phase
func (p *RemoveNodes) Title() string {
	return "Remove nodes"
}

// Run removes all nodes from swarm that are labeled and not part of the current config
func (p *RemoveNodes) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	nodeIDs, err := p.labelCurrentNodes(config, swarmLeader)
	if err != nil {
		return err
	}
	swarmIDs, err := p.swarmNodeIDs(swarmLeader)
	if err != nil {
		return err
	}
	for _, nodeID := range swarmIDs {
		if !util.StringSliceContains(nodeIDs, nodeID) && p.isManagedByUs(swarmLeader, nodeID) {
			err = p.removeNode(swarmLeader, nodeID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *RemoveNodes) labelCurrentNodes(config *api.ClusterConfig, swarmLeader *api.Host) ([]string, error) {
	nodeIDs := []string{}
	for _, h := range config.Spec.Hosts {
		nodeID, err := swarm.NodeID(h)
		if err != nil {
			return []string{}, err
		}
		labelCmd := swarmLeader.Configurer.DockerCommandf("node update --label-add com.mirantis.launchpad.managed=true %s", nodeID)
		err = swarmLeader.Exec(labelCmd)
		if err != nil {
			return []string{}, fmt.Errorf("Failed to label node %s (%s)", h.Address, nodeID)
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	return nodeIDs, nil
}

func (p *RemoveNodes) swarmNodeIDs(swarmLeader *api.Host) ([]string, error) {
	output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node ls --format="{{.ID}}"`))
	if err != nil {
		log.Errorln(output)
		return []string{}, err
	}
	return strings.Split(output, "\n"), nil
}

func (p *RemoveNodes) removeNode(swarmLeader *api.Host, nodeID string) error {
	nodeAddr, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format {{.Status.Addr}}`, nodeID))
	if err != nil {
		return err
	}
	log.Infof("%s: removing orphan node %s", swarmLeader.Address, nodeAddr)
	nodeRole, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format {{.Spec.Role}}`, nodeID))
	if err != nil {
		return err
	}
	if nodeRole == "manager" {
		log.Infof("%s: demoting orphan node %s", swarmLeader.Address, nodeAddr)
		err = swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf(`node demote %s`, nodeID))
		if err != nil {
			return err
		}
		log.Infof("%s: orphan node %s demoted", swarmLeader.Address, nodeAddr)
	}

	log.Infof("%s: draining orphan node %s", swarmLeader.Address, nodeAddr)
	drainCmd := swarmLeader.Configurer.DockerCommandf("node update --availability drain %s", nodeID)
	err = swarmLeader.Exec(drainCmd)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	log.Infof("%s: orphan node %s drained", swarmLeader.Address, nodeAddr)

	removeCmd := swarmLeader.Configurer.DockerCommandf("node rm --force %s", nodeID)
	err = swarmLeader.Exec(removeCmd)
	if err != nil {
		return err
	}
	log.Infof("%s: removed orphan node %s", swarmLeader.Address, nodeAddr)
	return nil
}

func (p *RemoveNodes) isManagedByUs(swarmLeader *api.Host, nodeID string) bool {
	labels, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`node inspect %s --format="{{json .Spec.Labels}}"`, nodeID))
	if err != nil {
		return false
	}
	return strings.Contains(labels, `"com.mirantis.launchpad.managed":"true"`)
}
