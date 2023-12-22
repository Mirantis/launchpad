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

	if config.Spec.MSR.LoadBalancerURL != "" {
		msrURL = "https://" + config.Spec.MSR.LoadBalancerURL + "/"
	} else {
		kc, _, err := mke.KubeAndHelmFromConfig(config)
		if err != nil {
			return "", fmt.Errorf("failed to get kube client: %s", err)
		}

		url, err := kc.MSRURL(context.Background(), config.Spec.MSR.MSR3Config.Name)
		if err != nil {
			return "", fmt.Errorf("failed to build MSR URL from Kubernetes services: %s", err)
		} else {
			msrURL = url.String()
		}
	}

	return msrURL, nil
}
