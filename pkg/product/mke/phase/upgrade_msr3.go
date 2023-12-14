package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// UpgradeMSR is the phase implementation for running the actual msr upgrade container.
type UpgradeMSR3 struct {
	phase.Analytics
	phase.CleanupDisabling
	MSR3Phase
}

// Title prints the phase title.
func (p *UpgradeMSR3) Title() string {
	return "Upgrade MSR3 components"
}

// ShouldRun should return true only when there is an upgrade to be performed.
func (p *UpgradeMSR3) ShouldRun() bool {
	h := p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR3() && h.MSRMetadata != nil && h.MSRMetadata.InstalledVersion != p.Config.Spec.MSR.Version
}

// Run updates the MSR CRD to the desired version tag and then applies the CRD.
func (p *UpgradeMSR3) Run() error {
	h := p.Config.Spec.MSRLeader()

	err := p.Config.Spec.CheckMKEHealthRemote(h)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", h)
	}

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR.Version == h.MSRMetadata.InstalledVersion {
		log.Infof("%s: msr cluster already at version %s, will not modify version within MSR CRD", h, p.Config.Spec.MSR.Version)
		return nil
	}

	p.Config.Spec.MSR.MSR3Config.MSR.Spec.Image.Tag = p.Config.Spec.MSR.Version

	if err := msr3.ApplyCRD(context.Background(), &p.Config.Spec.MSR.MSR3Config.MSR, p.kube); err != nil {
		return err
	}

	msrMeta, err := msr3.CollectFacts(context.Background(), p.Config.Spec.MSR.MSR3Config.Name, p.kube, p.helm)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing msr details: %s", h, err.Error())
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return fmt.Errorf("%s: upgraded msr version: %s does not match intended upgrade version: %s", h, msrMeta.InstalledVersion, p.Config.Spec.MSR.Version)
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = h.MSRMetadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR.Version
	h.MSRMetadata = msrMeta

	return nil
}
