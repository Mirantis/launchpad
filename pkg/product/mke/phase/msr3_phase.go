package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	MSRNodeSelector = "node-role.kubernetes.io/msr"
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

// ConvertSpecToUnstructured converts the MSR CRD defined in config to an
// unstructured object that can be used by KubeClient.
func (p *MSR3Phase) ConvertSpecToUnstructured() (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{Object: make(map[string]interface{})}

	err := mapstructure.Decode(p.Config.Spec.MSR.MSR3Config.MSR, &obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MSR CR into unstructured object: %w", err)
	}

	return obj, nil
}

// ApplyCRD applies the MSR CRD defined in p.Config.Spec.MSR.MSR3Config to
// the cluster then waits for it to be ready.
func (p *MSR3Phase) ApplyCRD(ctx context.Context) error {
	// Append the NodeSelector for the MSR hosts if not already present.
	ns := p.Config.Spec.MSR.MSR3Config.MSR.Spec.NodeSelector

	if _, ok := ns[MSRNodeSelector]; !ok {
		ns[MSRNodeSelector] = "true"
	}

	obj, err := p.ConvertSpecToUnstructured()
	if err != nil {
		return err
	}

	if err := p.kube.ApplyMSRCR(ctx, obj); err != nil {
		return err
	}

	return p.kube.WaitForMSRCRReady(ctx, obj)
}
