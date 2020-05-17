package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

// GatherUcpFacts collects facts about possibly existing UCP setup
type GatherUcpFacts struct{}

// Title for the phase
func (p *GatherUcpFacts) Title() string {
	return "Gather UCP facts"
}

// Run collects the facts from swarm leader
func (p *GatherUcpFacts) Run(conf *config.ClusterConfig) error {
	swarmLeader := conf.Controllers()[0]
	ucpMeta, err := util.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	conf.Ucp.Metadata = ucpMeta
	if ucpMeta.Installed {
		log.Infof("%s: UCP has version %s", swarmLeader.Address, ucpMeta.InstalledVersion)
	} else {
		log.Infof("%s: UCP is not installed", swarmLeader.Address)
	}

	return nil
}
