package msr3

import (
	"context"
	"errors"
	"fmt"

	msrv1 "github.com/Mirantis/msr-operator/api/v1"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// CollectFacts gathers the current status of the installed MSR3 setup.
func CollectFacts(ctx context.Context, msrName string, kc *kubeclient.KubeClient, rc dynamic.ResourceInterface, h *helm.Helm, options ...kubeclient.WaitOption) (*api.MSRMetadata, error) {
	obj, err := kc.GetMSRCR(ctx, msrName, rc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("MSR CR: %s not found: %s", msrName, err)
			return &api.MSRMetadata{Installed: false}, nil
		}
		return nil, err
	}

	if err := kc.WaitForMSRCRReady(ctx, obj, rc, options...); err != nil {
		// If we've failed to validate the MSR CR is ready then we cannot
		// reliably determine whether it is installed or not.
		return nil, err
	}

	version, found, err := unstructured.NestedString(obj.Object, "spec", "image", "tag")
	if !found || version == "" {
		err = errors.New("unable to determine version from found MSR: spec.image.tag not populated")
	}
	if err != nil {
		return nil, fmt.Errorf("unable to determine version from found MSR: %w", err)
	}

	releases, err := h.List(ctx, constant.InstalledDependenciesFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list helm releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("failed to find any installed helm dependencies")
	}

	installedDeps := make(map[string]helm.ReleaseDetails)

	for _, rel := range releases {
		installedDeps[rel.Name] = helm.ReleaseDetails{
			ReleaseName: rel.Name,
			ChartName:   rel.Chart.Name(),
			Version:     rel.Chart.Metadata.Version,
			Installed: func() bool {
				return rel.Info.Status.String() == string(release.StatusDeployed)
			}(),
		}
	}

	return &api.MSRMetadata{
		Installed:        true,
		InstalledVersion: version,
		MSR3: api.MSR3Metadata{
			InstalledDependencies: installedDeps,
		},
	}, nil
}

// ApplyCRD applies the MSR CRD to the cluster then waits for it to be ready.
func ApplyCRD(ctx context.Context, msr *msrv1.MSR, kc *kubeclient.KubeClient) error {
	obj, err := kubeclient.DecodeIntoUnstructured(msr)
	if err != nil {
		return fmt.Errorf("failed to decode MSR CR: %w", err)
	}

	rc, err := kc.GetMSRResourceClient()
	if err != nil {
		return fmt.Errorf("failed to get resource client for MSR CR: %w", err)
	}

	if err := kc.ApplyMSRCR(ctx, obj, rc); err != nil {
		return err
	}

	return kc.WaitForMSRCRReady(ctx, obj, rc)
}
