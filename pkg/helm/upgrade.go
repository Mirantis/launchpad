package helm

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	DefaultTimeout = time.Second * 300
)

// ReleaseDetails contains details about a Helm chart release.
type ReleaseDetails struct {
	// ChartName is the name of the Helm chart.
	ChartName string `yaml:"chartName,omitempty"`
	// ReleaseName is the name of the Helm release.
	ReleaseName string `yaml:"releaseName,omitempty"`
	// RepoURL is the URL to the Helm repository.
	RepoURL string `yaml:"repoURL,omitempty"`
	// Version is the Helm Chart version.
	Version string `yaml:"version,omitempty"`
	// Values contains options for the Helm chart values.
	Values map[string]any `yaml:"values,omitempty"`
	// Installed is true if the chart is installed.
	Installed bool `yaml:"installed,omitempty"`
}

// Options to be used with Helm actions.
type Options struct {
	// ReleaseDetails contains details about a Helm chart release.
	ReleaseDetails
	// ReuseValues will re-use the user's last supplied values.
	ReuseValues bool
	// Wait determines whether the wait operation should be performed after the upgrade is requested.
	Wait bool
	// Atomic, if true, will roll back on failure.
	Atomic bool
	// Timeout is the timeout for upgrade.
	Timeout *time.Duration
}

// Upgrade performs a `helm upgrade --install` with a subset of options.
func (h *Helm) Upgrade(ctx context.Context, opts *Options) (rel *release.Release, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to upgrade Helm release %q: %w", opts.ReleaseName, err)
		}
	}()

	// Create a copy of config & settings so that we don't
	// pass in a pointer to the struct's config and settings.
	cfg := h.config
	cfg.Capabilities.KubeVersion = chartutil.KubeVersion{Major: "1", Minor: "22", Version: "1.22.0"}
	settings := h.settings

	u := action.NewUpgrade(&cfg)

	chartToUpgrade, err := getChart(u.ChartPathOptions, opts.ChartName, &settings)
	if err != nil {
		return nil, err
	}

	// Install the chart if it is not already installed.
	histClient := action.NewHistory(&cfg)
	histClient.Max = 1
	if _, err := histClient.Run(opts.ReleaseName); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			log.Infof("release %q not found, installing it now", opts.ReleaseName)
			return h.install(ctx, opts, opts.Values, chartToUpgrade)
		}

		return nil, fmt.Errorf("failed to retrieve release history for %q: %w", opts.ReleaseName, err)
	}

	log.Infof("release %q found using chart: %q, upgrading to version: %q", opts.ReleaseName, opts.ChartName, opts.Version)

	u.Namespace = settings.Namespace()
	u.ReuseValues = opts.ReuseValues
	u.Wait = opts.Wait
	u.Atomic = opts.Atomic
	u.Version = opts.Version
	if opts.Timeout != nil {
		u.Timeout = *opts.Timeout
	}

	release, err := u.RunWithContext(ctx, opts.ReleaseName, chartToUpgrade, opts.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade Helm release %q: %w", opts.ReleaseName, err)
	}

	return release, nil
}

// install is ran as part of the Upgrade process when the chart is not
// yet installed.  Our Upgrade is essentially the equivalent of
// 'helm upgrade --install'.
func (h *Helm) install(ctx context.Context, opts *Options, vals map[string]interface{}, chartToInstall *chart.Chart) (rel *release.Release, err error) {
	cfg := h.config
	settings := h.settings

	installAction := action.NewInstall(&cfg)

	if opts.Timeout != nil {
		installAction.Timeout = *opts.Timeout
	}

	installAction.Namespace = settings.Namespace()
	installAction.ReleaseName = opts.ReleaseName
	installAction.Version = opts.Version
	installAction.Atomic = opts.Atomic
	installAction.Wait = opts.Wait

	release, err := installAction.RunWithContext(ctx, chartToInstall, vals)
	if err != nil {
		return nil, fmt.Errorf("failed to install Helm release %q: %w", opts.ReleaseName, err)
	}

	return release, nil
}

func getChart(chartPathOptions action.ChartPathOptions, chartName string, settings *cli.EnvSettings) (*chart.Chart, error) {
	chartPath, err := chartPathOptions.LocateChart(chartName, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %q: %w", chartName, err)
	}

	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	return loadedChart, nil
}
