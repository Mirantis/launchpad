package phase

import (
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/phase"
)

// MSR3Phase only runs when the config includes MSR hosts which are configured
// to use MSR3.
type MSR3Phase struct {
	phase.BasicPhase

	helm *helm.Helm
	kube *kubeclient.KubeClient
}

// ShouldRun default implementation for MSR phase returns true when the config
// has MSR nodes and MSR3 configuration.
func (p *MSR3Phase) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR3()
}
