package phase

import (
	"context"
	"fmt"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// getMSRURL returns the URL for the MSR admin UI.
func getMSRURL(config *api.ClusterConfig) (string, error) {
	var msrURL string

	if config.Spec.MSR.V3.LoadBalancerURL != "" {
		msrURL = "https://" + config.Spec.MSR.V3.LoadBalancerURL + "/"
	} else {
		kc, _, err := mke.KubeAndHelmFromConfig(config)
		if err != nil {
			return "", fmt.Errorf("failed to get kube client: %s", err)
		}

		rc, err := kc.GetMSRResourceClient()
		if err != nil {
			return "", fmt.Errorf("failed to get resource client for MSR CR: %w", err)
		}

		url, err := kc.MSRURL(context.Background(), config.Spec.MSR.V3.CRD.GetName(), rc)
		if err != nil {
			return "", fmt.Errorf("failed to build MSR URL from Kubernetes services: %s", err)
		} else {
			msrURL = url.String()
		}
	}

	return msrURL, nil
}
