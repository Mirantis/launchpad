package phase

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	log "github.com/sirupsen/logrus"
)

// Before phase implementation does all the prep work we need for the hosts
type Before struct {
	Analytics
}

// Title for the phase
func (p *Before) Title() string {
	return "Run Before Hooks"
}

// Run does all the prep work on the hosts in parallel
func (p *Before) Run(config *api.ClusterConfig) error {
	hosts := config.Spec.Hosts.Filter(func(h *api.Host) bool {
		return len(h.Before) > 0
	})
	return hosts.ParallelEach(func(h *api.Host) error {
		for _, cmd := range h.Before {
			log.Infof("%s: Executing: %s", h.Address, cmd)
			output, err := h.ExecWithOutput(cmd)
			if err != nil {
				log.Errorf("%s: %s", h.Address, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h.Address)))
				return err
			}
			log.Infof("%s: %s", h.Address, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h.Address)))
		}
		return nil
	})
}
