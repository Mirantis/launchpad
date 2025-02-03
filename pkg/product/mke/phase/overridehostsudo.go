package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/alessio/shellescape"
	"github.com/mattn/go-shellwords"
	log "github.com/sirupsen/logrus"
)

// OverrideHostSudo of the host if it has an override in the config.
type OverrideHostSudo struct {
	phase.Analytics
	phase.HostSelectPhase

	overrideHosts api.Hosts
}

// Title for the phase.
func (p *OverrideHostSudo) Title() string {
	return "Override the host sudo"
}

// ShouldRun should return true only when there is a host with an overridet.
func (p *OverrideHostSudo) ShouldRun() bool {
	for _, h := range p.Hosts {
		if h.SudoOverride {
			p.overrideHosts = append(p.overrideHosts, h)
		}
	}
	return len(p.overrideHosts) > 0
}

// Run the phase.
func (p *OverrideHostSudo) Run() error {
	err := p.Hosts.ParallelEach(func(h *api.Host) error {
		if h.SudoOverride {
			log.Warnf("%s: overriding sudo for host", h)
			h.SetSudofn(sudoSudo)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to override sudo on hosts: %w", err)
	}
	return nil
}

// @see https://github.com/k0sproject/rig/blob/release-0.x/connection.go#L253
func sudoSudo(cmd string) string {
	parts, err := shellwords.Parse(cmd)
	if err != nil {
		return "sudo -- " + cmd
	}

	var idx int
	for i, p := range parts {
		if strings.Contains(p, "=") {
			idx = i + 1
			continue
		}
		break
	}

	if idx == 0 {
		return "sudo -- " + cmd
	}

	for i, p := range parts {
		parts[i] = shellescape.Quote(p)
	}

	return fmt.Sprintf("sudo %s -- %s", strings.Join(parts[0:idx], " "), strings.Join(parts[idx:], " "))
}
