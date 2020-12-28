package k0s

import (
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	k0s "github.com/Mirantis/mcc/pkg/product/k0s/phase"
)

// Apply installs k0s on the desired host
func (p *K0s) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&k0s.GatherFacts{},
		&k0s.PrepareConfig{},
		&k0s.InstallK0s{},
		&k0s.StartK0s{},
		&common.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return err
	}
	return nil
}
