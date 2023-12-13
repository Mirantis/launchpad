package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
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
	phase.KubePhase

	ctx context.Context
}

// Prepare prepares the MSR3Phase by populating the context.
func (p *MSR3Phase) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	p.ctx = context.Background()

	return nil
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

	if err := p.Kube.ApplyMSRCR(ctx, obj); err != nil {
		return err
	}

	return p.Kube.WaitForMSRCRReady(ctx, obj)
}
