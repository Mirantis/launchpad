package phase

import (
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/phase"
)

// MSRPhase only runs when the config includes MSR hosts which are configured
// to use MSR (v2).
type MSRPhase struct {
	phase.BasicPhase
}

// MSR3Phase only runs when the config includes MSR hosts which are configured
// to use MSR3.
type MSR3Phase struct {
	phase.BasicPhase

	helm *helm.Helm
	kube *kubeclient.KubeClient
}
