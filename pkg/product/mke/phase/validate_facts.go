package phase

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/swarm"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

var ErrFactsArentValid = errors.New("validation failed")

// ValidateFacts phase implementation to validate facts from config and collected metadata.
type ValidateFacts struct {
	phase.Analytics
	phase.BasicPhase
	Force bool
}

// Title for the phase.
func (p *ValidateFacts) Title() string {
	return "Validate Facts"
}

// Run validate configuration facts.
func (p *ValidateFacts) Run() error {
	if p.Config.Spec.MKE.Version == "" {
		return errors.Join(ErrFactsArentValid, fmt.Errorf("MKE spec is required (spec.mke.version)"))
	}

	if !p.Config.Spec.MKE.InstallFlags.Include("--san") {
		p.populateSan()
	}

	_ = p.Config.Spec.Hosts.Each(func(h *mkeconfig.Host) error {
		if h.Configurer != nil && h.Configurer.SELinuxEnabled(h) {
			h.DaemonConfig["selinux-enabled"] = true
			log.Infof("%s: adding 'selinux-enabled=true' to host container runtime config", h)
		}

		return nil
	})

	if err := p.validateMKEVersionJump(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return errors.Join(ErrFactsArentValid, err)
		}
	}

	if err := p.validateMSRVersionJump(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return errors.Join(ErrFactsArentValid, err)
		}
	}

	if err := p.validateDataPlane(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return errors.Join(ErrFactsArentValid, err)
		}
	}

	p.warnSwarmAddrPoolDivergence()

	if err := p.validatePodCIDR(); err != nil {
		return errors.Join(ErrFactsArentValid, err)
	}

	return nil
}

func (p *ValidateFacts) populateSan() {
	mgrs := p.Config.Spec.Managers()
	for _, h := range mgrs {
		f := fmt.Sprintf("--san=%s", h.Address())
		p.Config.Spec.MKE.InstallFlags.Add(f)
		log.Warnf("%s: added manager node's public address to mke installFlag SANs: %s", h, f)
	}
}

var errInvalidUpgradePath = errors.New("invalid upgrade path")

// validateMKEVersionJump validates MKE upgrade path.
func (p *ValidateFacts) validateMKEVersionJump() error {
	if p.Config.Spec.MKE.Metadata == nil || !p.Config.Spec.MKE.Metadata.Installed || p.Config.Spec.MKE.Metadata.InstalledVersion == "" {
		return nil
	}
	installedMKE, err := version.NewVersion(p.Config.Spec.MKE.Metadata.InstalledVersion)
	if err != nil {
		return fmt.Errorf("can't parse installed MKE version: %w", err)
	}
	targetMKE, err := version.NewVersion(p.Config.Spec.MKE.Version)
	if err != nil {
		return fmt.Errorf("can't parse target MKE version: %w", err)
	}

	if mke.VersionGreaterThan(installedMKE, targetMKE) {
		return fmt.Errorf("%w: can't downgrade MKE %s to %s", errInvalidUpgradePath, installedMKE, targetMKE)
	}

	installedSegments := installedMKE.Segments()
	targetSegments := targetMKE.Segments()

	// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
	if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
		return fmt.Errorf("%w: can't upgrade MKE directly from %s to %s - need to upgrade to %d.%d first", errInvalidUpgradePath, installedMKE, targetMKE, installedSegments[0], installedSegments[1]+1)
	}
	return nil
}

// validateMSRVersionJump validates MSR upgrade path.
func (p *ValidateFacts) validateMSRVersionJump() error {
	msrLeader := p.Config.Spec.MSRLeader()
	if p.Config.Spec.MSR != nil && msrLeader.MSRMetadata != nil && msrLeader.MSRMetadata.Installed && msrLeader.MSRMetadata.InstalledVersion != "" {
		installedMSR, err := version.NewVersion(msrLeader.MSRMetadata.InstalledVersion)
		if err != nil {
			return fmt.Errorf("can't parse installed MSR version: %w", err)
		}
		targetMSR, err := version.NewVersion(p.Config.Spec.MSR.Version)
		if err != nil {
			return fmt.Errorf("can't parse target MSR version: %w", err)
		}

		if mke.VersionGreaterThan(installedMSR, targetMSR) {
			return fmt.Errorf("%w: can't downgrade MSR %s to %s", errInvalidUpgradePath, installedMSR, targetMSR)
		}

		installedSegments := installedMSR.Segments()
		targetSegments := targetMSR.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("%w: can't upgrade MSR directly from %s to %s - need to upgrade to %d.%d first", errInvalidUpgradePath, installedMSR, targetMSR, installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

var errInvalidDataPlane = errors.New("invalid data plane settings")

// validateDataPlane checks if the calico data plane would get changed (VXLAN <-> VPIP).
func (p *ValidateFacts) validateDataPlane() error {
	log.Debug("validating data plane settings")

	idx := p.Config.Spec.MKE.InstallFlags.Index("--calico-vxlan")
	if idx < 0 {
		return nil
	}

	val := p.Config.Spec.MKE.InstallFlags.GetValue("--calico-vxlan")
	var valB bool
	if val == "" {
		valB = true
	} else {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("can't parse --calico-vxlan value: %w", err)
		}
		valB = v
	}

	// User has explicitly defined --calico-vxlan=false but there is a windows host in the config
	if !valB {
		if p.Config.Spec.Hosts.Include(func(h *mkeconfig.Host) bool { return h.IsWindows() }) {
			return fmt.Errorf("%w: calico IPIP can't be used on Windows", errInvalidDataPlane)
		}

		log.Debug("no windows hosts found")
	}

	if p.Config.Spec.MKE.Metadata == nil || !p.Config.Spec.MKE.Metadata.Installed {
		log.Debug("no existing MKE installation")
		return nil
	}

	// User has explicitly defined --calico-vxlan=false but there is already a calico with vxlan
	if p.Config.Spec.MKE.Metadata.VXLAN {
		log.Debug("mke has been installed with calico + vxlan")
		if !valB {
			return fmt.Errorf("%w: calico configured with VXLAN, can't automatically change to IPIP", errInvalidDataPlane)
		}
	} else {
		log.Debug("mke has been installed with calico + vpip")
		// User has explicitly defined --calico-vxlan=true but there is already a calico with ipip
		if valB {
			return fmt.Errorf("%w: calico configured with IPIP, can't automatically change to VXLAN", errInvalidDataPlane)
		}
	}

	log.Debug("data plane settings check passed")

	return nil
}

var errInvalidPodCIDR = errors.New("invalid pod CIDR configuration")

// swarmAddrPools returns the overlay address pools that --pod-cidr must not
// overlap, and whether they describe a swarm that already exists.
//
// An existing swarm is authoritative. Its pool is fixed at creation and the
// InitSwarm phase discards mcr.swarmInstallFlags on such a cluster, so the
// configured value describes nothing there. Before a swarm exists the configured
// value is what swarm init will apply, and so is the pool to validate against.
func (p *ValidateFacts) swarmAddrPools() (pools []string, fromExistingSwarm bool) {
	if p.Config.Spec.MCR.Metadata != nil && len(p.Config.Spec.MCR.Metadata.SwarmDefaultAddrPool) > 0 {
		return p.Config.Spec.MCR.Metadata.SwarmDefaultAddrPool, true
	}

	// docker swarm init accepts --default-addr-pool repeated, so keep every pool.
	if configured := p.Config.Spec.MCR.SwarmInstallFlags.GetValues("--default-addr-pool"); len(configured) > 0 {
		return configured, false
	}

	return []string{swarm.DefaultAddrPoolFallback}, false
}

// warnSwarmAddrPoolDivergence reports an mcr.swarmInstallFlags --default-addr-pool
// setting that describes something other than the running cluster. The pool is
// fixed when the swarm is created, so on an existing cluster the setting is inert
// and the configuration quietly stops matching the infrastructure.
//
// Nothing is reported unless a pool is both explicitly configured and known to
// differ from the live one, so the common case of a cluster that never set the
// flag stays silent.
func (p *ValidateFacts) warnSwarmAddrPoolDivergence() {
	if p.Config.Spec.MCR.Metadata == nil || len(p.Config.Spec.MCR.Metadata.SwarmDefaultAddrPool) == 0 {
		return
	}

	configured := p.Config.Spec.MCR.SwarmInstallFlags.GetValues("--default-addr-pool")
	if len(configured) == 0 {
		return
	}

	live := p.Config.Spec.MCR.Metadata.SwarmDefaultAddrPool
	if slices.Equal(configured, live) {
		return
	}

	log.Warnf(
		"mcr.swarmInstallFlags sets --default-addr-pool %s but the existing swarm allocates overlay networks from %s. "+
			"A swarm's address pool is fixed when the swarm is created and cannot be changed on a running cluster, so this "+
			"setting has no effect here and the cluster no longer matches its configuration. Changing the pool requires "+
			"dissolving and re-creating the swarm, which destroys every overlay network and service",
		strings.Join(configured, ","), strings.Join(live, ","),
	)
}

// validatePodCIDR checks that --pod-cidr in mke.installFlags does not overlap the
// Swarm overlay address pool. Overlapping CIDRs cause the Docker daemon to restart
// into a broken network state during MKE bootstrap, which silently drops the SSH
// connection and produces a connection timeout after 20+ minutes.
//
// The conflict is fatal only while it is still avoidable, which means before the
// swarm exists. On an existing swarm the pool cannot be changed, so the conflict
// is reported and the run continues rather than blocking upgrades of clusters that
// are already running this way. See PRODENG-3642.
func (p *ValidateFacts) validatePodCIDR() error {
	podCIDRStr := p.Config.Spec.MKE.InstallFlags.GetValue("--pod-cidr")
	if podCIDRStr == "" {
		return nil
	}

	_, podNet, err := net.ParseCIDR(podCIDRStr)
	if err != nil {
		return fmt.Errorf("%w: cannot parse --pod-cidr %q: %w", errInvalidPodCIDR, podCIDRStr, err)
	}

	swarmPools, fromExistingSwarm := p.swarmAddrPools()
	overlapping := false

	for _, swarmPoolStr := range swarmPools {
		_, swarmNet, err := net.ParseCIDR(swarmPoolStr)
		if err != nil {
			if fromExistingSwarm {
				// Reported by the daemon rather than written by the user, so
				// there is nothing in the configuration to correct.
				log.Warnf("cannot parse the overlay address pool %q reported by the existing swarm: %s", swarmPoolStr, err.Error())
				continue
			}

			return fmt.Errorf("%w: cannot parse Swarm address pool %q: %w", errInvalidPodCIDR, swarmPoolStr, err)
		}

		if !swarmNet.Contains(podNet.IP) && !podNet.Contains(swarmNet.IP) {
			continue
		}

		if !fromExistingSwarm {
			return fmt.Errorf(
				"%w: --pod-cidr %s overlaps with the Swarm overlay address pool %s; "+
					"choose a non-overlapping range or set mcr.swarmInstallFlags --default-addr-pool to a non-conflicting pool",
				errInvalidPodCIDR, podCIDRStr, swarmPoolStr,
			)
		}

		overlapping = true

		log.Warnf(
			"--pod-cidr %s overlaps the overlay address pool %s of the existing swarm, which can leave the container "+
				"runtime with a broken network configuration during MKE bootstrap. The pool is fixed when a swarm is "+
				"created, so mcr.swarmInstallFlags cannot resolve this on a running cluster: either choose a "+
				"non-overlapping --pod-cidr, or dissolve and re-create the swarm with a non-overlapping pool",
			podCIDRStr, swarmPoolStr,
		)
	}

	if !overlapping {
		log.Debugf("pod CIDR %s does not overlap with any Swarm pool %v", podCIDRStr, swarmPools)
	}

	return nil
}
