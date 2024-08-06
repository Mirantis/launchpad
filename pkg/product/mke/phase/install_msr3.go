package phase

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// InstallOrUpgradeMSR3 deploys an MSR Custom Resource using the CRD provided
// within config.  It handles both the Install and Upgrade phases for MSR3.
type InstallOrUpgradeMSR3 struct {
	phase.Analytics
	phase.CleanupDisabling
	phase.KubernetesPhase
}

// Title prints the phase title.
func (p *InstallOrUpgradeMSR3) Title() string {
	return "Configuring MSR Custom Resource"
}

// Prepare collects the hosts and labels them with the MSR role via the
// Kubernetes client so that they can be used as NodeSelector in the MSR CR.
func (p *InstallOrUpgradeMSR3) Prepare(_ interface{}) error {
	managers := p.Config.Spec.Managers()

	if err := p.Config.Spec.CheckMKEHealthRemote(managers); err != nil {
		return fmt.Errorf("failed to health check mke, try to set `--ucp-url` installation flag and check connectivity: %w", err)
	}

	var err error
	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return fmt.Errorf("failed to get kube and helm clients: %w", err)
	}

	return nil
}

// ShouldRun should return true only if MSR3 config is present.
func (p *InstallOrUpgradeMSR3) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR3()
}

// Run deploys an MSR CR to the cluster.
func (p *InstallOrUpgradeMSR3) Run() error {
	ctx := context.Background()

	if err := p.Kube.ValidateMSROperatorReady(ctx); err != nil {
		return fmt.Errorf("failed to validate msr-operator is ready: %w", err)
	}

	msr := p.Config.Spec.MSR3.CRD

	// Configure an imagePullSecret if necessary.
	registryUsername := os.Getenv("REGISTRY_USERNAME")
	registryPassword := os.Getenv("REGISTRY_PASSWORD")
	registryURL, _, err := unstructured.NestedString(p.Config.Spec.MSR3.CRD.Object, "spec", "image", "registry")
	if err != nil {
		return fmt.Errorf("failed to get registry URL from MSR CRD: %w", err)
	}

	if registryUsername != "" && registryPassword != "" {
		registryAuth := kubeclient.DockerAuth{
			Username: os.Getenv("REGISTRY_USERNAME"),
			Password: os.Getenv("REGISTRY_PASSWORD"),
			URL:      "https://" + registryURL,
		}

		if err := p.Kube.CreateImagePullSecret(context.Background(), registryAuth); err != nil {
			return fmt.Errorf("failed to patch service account with image pull secrets: %w", err)
		}

		// Append our created secret onto the existing pull secrets.  Note: The
		// yaml tag is "pullSecret" not "pullSecrets", so the field name here
		// is intentional.
		if err := appendImagePullSecret(msr, "spec", "image", "pullSecret"); err != nil {
			return fmt.Errorf("failed to append MSR image pull secret to MSR CRD: %w", err)
		}
		if err := appendImagePullSecret(msr, "spec", "enzi", "pullSecret"); err != nil {
			return fmt.Errorf("failed to append Enzi image pull secret to MSR CRD: %w", err)
		}
		if err := appendImagePullSecret(msr, "spec", "rethinkdb", "dockerImage", "pullSecret"); err != nil {
			return fmt.Errorf("failed to append RethinkDB image pull secret to MSR CRD: %w", err)
		}
	}

	// Ensure the postgresql.spec.volume.size field is sane, postgres-operator
	// doesn't default the Size field and is picky about the format.
	postgresVolumeSize, found, err := unstructured.NestedString(msr.Object, "spec", "postgresql", "volume", "size")
	if err != nil {
		return fmt.Errorf("failed to get MSR spec.postgresql.volume.size: %w", err)
	}

	if !found || postgresVolumeSize == "" {
		if err := unstructured.SetNestedField(msr.Object, "20Gi", "spec", "postgresql", "volume", "size"); err != nil {
			return fmt.Errorf("failed to set MSR spec.postgresql.volume.size: %w", err)
		}
	}

	// Set the version tag to the desired MSR version specified in config.
	if err := unstructured.SetNestedField(msr.Object, p.Config.Spec.MSR3.Version, "spec", "image", "tag"); err != nil {
		return fmt.Errorf("failed to set MSR spec.image.tag: %w", err)
	}

	// Configure Nginx.DNSNames if a LoadBalancerURL is specified.
	if p.Config.Spec.MSR3.ShouldConfigureLB() {
		if err := unstructured.SetNestedStringSlice(msr.Object, []string{"nginx", "localhost", p.Config.Spec.MSR3.LoadBalancerURL}, "spec", "nginx", "dnsNames"); err != nil {
			return fmt.Errorf("failed to set MSR spec.nginx.dnsNames to include LoadBalancerURL: %q: %w", p.Config.Spec.MSR3.LoadBalancerURL, err)
		}
	}

	// TODO: Differentiate an upgrade from an install and set analytics
	// around that.
	if err := msr3.ApplyCRD(ctx, msr, p.Kube); err != nil {
		return fmt.Errorf("failed to apply MSR CRD: %w", err)
	}

	if p.Config.Spec.MSR3.ShouldConfigureLB() {
		if err := p.Kube.ExposeLoadBalancer(ctx, p.Config.Spec.MSR3.LoadBalancerURL); err != nil {
			log.Warnf("failed to expose MSR via LoadBalancer: %s", err)
		}
	}

	rc, err := p.Kube.GetMSRResourceClient()
	if err != nil {
		return fmt.Errorf("failed to get MSR resource client: %w", err)
	}

	msrMeta, err := msr3.CollectFacts(ctx, p.Config.Spec.MSR3.CRD.GetName(), p.Kube, rc, p.Helm)
	if err != nil {
		return fmt.Errorf("failed to collect MSR details: %w", err)
	}

	p.Config.Spec.MSR3.Metadata = msrMeta

	return nil
}

// appendImagePullSecret appends the Kubernetes pull secret to the
// existing pull secrets associated with fields in the MSR unstructured object.
func appendImagePullSecret(msr *unstructured.Unstructured, fields ...string) error {
	fieldsErrDetail := strings.Join(fields, ".")

	pullSecretSlice, found, err := unstructured.NestedSlice(msr.Object, fields...)
	if err != nil {
		return fmt.Errorf("failed to get MSR %s: %w", fieldsErrDetail, err)
	}

	if !found {
		log.Debugf("MSR %s not found, creating it", fieldsErrDetail)
	}

	// pullSecretSlice is an empty slice if error or not found, so we can
	// append to it either way.
	pullSecretSlice = append(pullSecretSlice, map[string]interface{}{"name": constant.KubernetesDockerRegistryAuthSecretName})
	if err := unstructured.SetNestedField(msr.Object, pullSecretSlice, fields...); err != nil {
		return fmt.Errorf("failed to set MSR %s: %w", fieldsErrDetail, err)
	}

	return nil
}
