package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"

	"github.com/Mirantis/mcc/pkg/constant"
)

type ErrReleaseNotFound struct {
	ReleaseName string
}

func (e ErrReleaseNotFound) Error() string {
	return fmt.Sprintf("release %q not found", e.ReleaseName)
}

type Helm struct {
	config   action.Configuration
	settings cli.EnvSettings
}

// New returns a configured Helm.  An MKE bundleDir which contains a kubeconfig
// file and a kubernetes namespace are required to scope the configured Helm
// client.
func New(bundleDir, namespace string) (*Helm, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	if err := os.Setenv("HELM_NAMESPACE", namespace); err != nil {
		return nil, fmt.Errorf("failed to scope Helm to namespace %q: %w", namespace, err)
	}

	kubeConfig := filepath.Join(bundleDir, constant.KubeConfigFile)

	cfg := action.Configuration{}
	if err := cfg.Init(
		kube.GetConfig(kubeConfig, "", namespace),
		namespace,
		os.Getenv("HELM_DRIVER"),
		log.Debugf,
	); err != nil {
		return nil, err
	}

	return &Helm{
		config:   cfg,
		settings: *settings,
	}, nil
}

// ChartNeedsUpgrade returns true if the chart version of the release is
// different from the one provided.  Helm Upgrade can be used to downgrade
// chart versions as well.
func (h *Helm) ChartNeedsUpgrade(ctx context.Context, releaseName string, newVersion *version.Version) (bool, error) {
	existingVersion, err := h.getChartVersion(ctx, releaseName)
	if err != nil {
		return false, err
	}

	if existingVersion.Equal(newVersion) {
		log.Debugf("release: %q is already at version: %q", releaseName, newVersion)
		return false, nil
	}

	log.Debugf("release: %q needs to match version: %q", releaseName, newVersion)

	return true, nil
}

// getChartVersion returns the version of the requested chart release if it
// is Deployed.
func (h *Helm) getChartVersion(ctx context.Context, releaseName string) (*version.Version, error) {
	releases, err := h.List(ctx, fmt.Sprintf("^%s$", releaseName))
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, ErrReleaseNotFound{ReleaseName: releaseName}
	}

	chartVersion := releases[0].Chart.Metadata.Version

	v, err := version.NewVersion(chartVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chart version %q: %w", chartVersion, err)
	}

	return v, nil
}

// List returns a list of Helm releases filtered by the provided filter.
func (h *Helm) List(ctx context.Context, filter string) ([]*release.Release, error) {
	l := action.NewList(&h.config)

	if filter != "" {
		l.Filter = filter
	}

	l.Deployed = true

	l.SetStateMask()

	releases, err := l.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list Helm releases: %w", err)
	}

	return releases, nil
}
