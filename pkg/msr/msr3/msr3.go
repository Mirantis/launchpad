package msr3

import (
	"context"
	"errors"
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

var (
	errSpecImageTagNotPopulated    = errors.New("unable to determine version from found MSR: spec.image.tag not populated")
	errNoHelmDependenciesInstalled = errors.New("failed to find any installed helm dependencies")
)

// CollectFacts gathers the current status of the installed MSR3 setup.
func CollectFacts(ctx context.Context, msrName string, kubeClient *kubeclient.KubeClient, resourceClient dynamic.ResourceInterface, helmClient *helm.Helm, options ...kubeclient.WaitOption) (api.MSR3Metadata, error) {
	obj, err := kubeClient.GetMSRCR(ctx, msrName, resourceClient)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("MSR CR: %s not found: %s", msrName, err)
			return api.MSR3Metadata{Installed: false}, nil
		}
		return api.MSR3Metadata{}, fmt.Errorf("failed to get MSR CR: %w", err)
	}

	// Check to see if the MSR CR exists and is ready for up to 30 seconds
	// before deeming it not installed as we will try to reconcile it again
	// later.
	if err := kubeClient.WaitForMSRCRReady(ctx, obj, resourceClient, options...); err != nil {
		// If we've failed to validate the MSR CR is ready then we cannot
		// reliably determine whether it is installed or not, so mark it as
		// not installed.
		log.Infof("Failed to determine if MSR CR: %s is ready: %s", msrName, err)
		return api.MSR3Metadata{Installed: false}, nil
	}

	version, found, err := unstructured.NestedString(obj.Object, "spec", "image", "tag")
	if !found || version == "" {
		return api.MSR3Metadata{}, errSpecImageTagNotPopulated
	}
	if err != nil {
		return api.MSR3Metadata{}, fmt.Errorf("unable to determine version from found MSR: %w", err)
	}

	releases, err := helmClient.List(constant.InstalledDependenciesFilter)
	if err != nil {
		return api.MSR3Metadata{}, fmt.Errorf("failed to list helm releases: %w", err)
	}

	if len(releases) == 0 {
		return api.MSR3Metadata{}, errNoHelmDependenciesInstalled
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

	return api.MSR3Metadata{
		Installed:             true,
		InstalledVersion:      version,
		InstalledDependencies: installedDeps,
	}, nil
}

// ApplyCRD applies the MSR CRD to the cluster then waits for it to be ready.
func ApplyCRD(ctx context.Context, msr *unstructured.Unstructured, kc *kubeclient.KubeClient) error {
	rc, err := kc.GetMSRResourceClient()
	if err != nil {
		return fmt.Errorf("failed to get resource client for MSR CR: %w", err)
	}

	if err := kc.ApplyMSRCR(ctx, msr, rc); err != nil {
		return fmt.Errorf("failed to apply MSR CR: %w", err)
	}

	if err := kc.WaitForMSRCRReady(ctx, msr, rc); err != nil {
		return fmt.Errorf("failed to wait for MSR CR to be ready: %w", err)
	}

	return nil
}

// GetMSRURL returns the URL for the MSR admin UI.
func GetMSRURL(config *api.ClusterConfig) (string, error) {
	if config.Spec.MSR3.LoadBalancerURL != "" {
		return "https://" + config.Spec.MSR3.LoadBalancerURL + "/", nil
	}

	kubeClient, _, err := mke.KubeAndHelmFromConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to get kube client: %w", err)
	}

	rc, err := kubeClient.GetMSRResourceClient()
	if err != nil {
		return "", fmt.Errorf("failed to get resource client for MSR CR: %w", err)
	}

	msrName := config.Spec.MSR3.CRD.GetName()

	url, err := kubeClient.MSRURL(context.Background(), msrName, rc)
	if err != nil {
		return "", fmt.Errorf("failed to build MSR URL from Kubernetes services: %w", err)
	}

	return url.String(), nil
}
