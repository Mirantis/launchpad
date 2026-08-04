package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/swarm"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

// UninstallMKE is the phase implementation for running MKE uninstall.
type UninstallMKE struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title.
func (p *UninstallMKE) Title() string {
	return "Uninstall MKE components"
}

// Run the installer container.
func (p *UninstallMKE) Run() error {
	leader := p.Config.Spec.SwarmLeader()
	if !p.Config.Spec.MKE.Metadata.Installed {
		log.Infof("%s: MKE is not installed, skipping", leader)
		return nil
	}

	uninstallFlags := commonconfig.Flags{"--id", swarm.ClusterID(leader), "--purge-config"}

	// Capture both output and error: the timeout message ("Uninstalling UCP
	// took too long") is emitted at error level by MKE and appears only in
	// the streamed output, not in the returned error (which only aggregates
	// fatal-level log lines from the bootstrapper).
	output, err := mke.Bootstrap("uninstall-ucp", *p.Config, mke.BootstrapOptions{OperationFlags: uninstallFlags, ExecOptions: []cmd.ExecOption{cmd.StreamOutput()}})
	if err != nil {
		// The uninstall-ucp bootstrapper deploys ucp-uninstall-agent as a global
		// Swarm service and waits (hardcoded ~2 minutes) for every node to report
		// back. On large clusters or hosts with cold image caches this deadline is
		// missed. When that happens, MKE itself recommends:
		//   1. Remove the stuck ucp-uninstall-agent service.
		//   2. Force every node to leave the swarm.
		// We implement that as an automatic fallback so that reset can continue
		// to MCR uninstall without leaving a broken cluster behind.
		if isUninstallTimeout(output) {
			log.Warnf("%s: uninstall-ucp timed out waiting for nodes; falling back to forced swarm dissolution", leader)
			if dissolveErr := dissolveSwarm(leader, p.Config.Spec.Hosts); dissolveErr != nil {
				return fmt.Errorf("%s: uninstall-ucp timed out and forced swarm dissolution failed: %w (original: %w)", leader, dissolveErr, err)
			}
			log.Infof("%s: swarm dissolved; continuing with MCR uninstall", leader)
		} else {
			return fmt.Errorf("%s: failed to run MKE uninstaller: %w", leader, err)
		}
	}

	managers := p.Config.Spec.Managers()
	_ = managers.ParallelEach(func(h *mkeconfig.Host) error {
		log.Infof("%s: removing ucp-controller-server-certs volume", h)
		if err := h.Exec(h.Configurer.DockerCommandf("volume rm --force ucp-controller-server-certs")); err != nil {
			log.Errorf("%s: failed to remove the volume: %v", h, err)
		}

		if err := h.Reboot(); err != nil {
			log.Errorf("%s: failed to reboot the host: %v", h, err)
		}
		return nil
	})

	workers := p.Config.Spec.WorkersAndMSRs()
	_ = workers.ParallelEach(func(h *mkeconfig.Host) error {
		if err := h.Reboot(); err != nil {
			log.Errorf("%s: failed to reboot the host: %v", h, err)
		}
		return nil
	})

	return nil
}

// isUninstallTimeout returns true when the streamed output from the
// uninstall-ucp bootstrapper contains the well-known node-acknowledgement
// timeout message. MKE emits this at error level (not fatal), so it appears
// only in Bootstrap's output string, not in the returned error value.
func isUninstallTimeout(output string) bool {
	return strings.Contains(output, "Uninstalling UCP took too long")
}

// dissolveSwarm forcibly tears down the Swarm cluster when uninstall-ucp
// cannot do so cleanly. It follows the recovery steps documented by MKE:
//
//  1. Remove the stuck ucp-uninstall-agent / ucp-uninstall-agent-win services
//     from the swarm leader (best-effort; they may already be gone).
//  2. Force all non-leader nodes to leave the swarm in parallel.
//  3. Force the leader to leave last.
//
// Errors from individual nodes are logged as warnings so that a single
// unresponsive host does not prevent the rest of the cluster from being torn
// down. Only the leader's final leave is treated as a hard failure.
func dissolveSwarm(leader *mkeconfig.Host, hosts mkeconfig.Hosts) error {
	// Step 1: remove the stuck uninstall-agent services (best-effort).
	for _, svc := range []string{"ucp-uninstall-agent", "ucp-uninstall-agent-win"} {
		log.Infof("%s: removing stuck service %s", leader, svc)
		if err := leader.Exec(leader.Configurer.DockerCommandf("service rm %s", svc)); err != nil {
			log.Debugf("%s: service rm %s: %v (may already be removed)", leader, svc, err)
		}
	}

	// Step 2: force all non-leader nodes to leave the swarm.
	nonLeaders := hosts.Filter(func(h *mkeconfig.Host) bool { return h != leader })
	_ = nonLeaders.ParallelEach(func(h *mkeconfig.Host) error {
		log.Infof("%s: force-leaving swarm", h)
		if err := h.Exec(h.Configurer.DockerCommandf("swarm leave --force")); err != nil {
			log.Warnf("%s: swarm leave --force failed: %v", h, err)
		}
		return nil // continue regardless; errors are warnings only
	})

	// Step 3: leader leaves last so it can still reach the other nodes above.
	log.Infof("%s: force-leaving swarm (leader)", leader)
	if err := leader.Exec(leader.Configurer.DockerCommandf("swarm leave --force")); err != nil {
		return fmt.Errorf("swarm leader failed to leave: %w", err)
	}

	return nil
}
