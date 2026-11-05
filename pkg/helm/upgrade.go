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

// errVerifyWithoutKeyring is returned when provenance verification is
// requested without a keyring to verify against.
var errVerifyWithoutKeyring = errors.New("provenance verification requires a keyring: set Keyring to the path of the chart signer's public keyring")

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
	// Verify enables Helm provenance verification: the chart's .prov
	// signature is checked against Keyring before the chart is loaded.
	//
	// It defaults to false, deliberately.  Nothing in this repository ships
	// or references a keyring, and the charts these options are used with are
	// pulled from OCI registries and repositories that publish no .prov
	// files, so defaulting Verify to true would fail every chart resolution
	// on a missing provenance file or a missing keyring rather than make any
	// chart more trustworthy.  A caller that has a signed chart and the
	// signer's public keyring opts in by setting Verify and Keyring together;
	// setting Verify without Keyring is rejected rather than silently
	// verifying against nothing.
	Verify bool
	// Keyring is the path to the public keyring used for provenance
	// verification.  It is only consulted when Verify is true.
	Keyring string
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

	upgrade := action.NewUpgrade(&cfg)

	// The chart is resolved and loaded exactly once, here, and the loaded
	// chart is what both the upgrade and the install path hand to Helm.  Any
	// value set on an action's ChartPathOptions after this point therefore
	// has no bearing on which chart is used, so the pinning options have to
	// be applied before LocateChart runs.
	if err := applyChartPathOptions(&upgrade.ChartPathOptions, opts); err != nil {
		return nil, err
	}

	chartToUpgrade, err := getChart(upgrade.ChartPathOptions, opts.ChartName, &settings)
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

	upgrade.Namespace = settings.Namespace()
	upgrade.ReuseValues = opts.ReuseValues
	upgrade.Wait = opts.Wait
	upgrade.Atomic = opts.Atomic
	if opts.Timeout != nil {
		upgrade.Timeout = *opts.Timeout
	}

	release, err := upgrade.RunWithContext(ctx, opts.ReleaseName, chartToUpgrade, opts.Values)
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
	// Keep the install action's chart source consistent with the options the
	// chart was actually resolved with in Upgrade.  The chart handed to
	// RunWithContext below is already loaded, so these values do not select
	// it a second time; they must be correct at LocateChart, which is where
	// Upgrade applies them.
	if err := applyChartPathOptions(&installAction.ChartPathOptions, opts); err != nil {
		return nil, err
	}
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

// applyChartPathOptions pins the chart resolution options on an action's
// ChartPathOptions.  The fields are assigned individually rather than by
// replacing the struct: action.NewInstall and action.NewUpgrade populate an
// unexported registry client on ChartPathOptions, and overwriting it would
// break resolution of oci:// charts.
func applyChartPathOptions(chartPathOptions *action.ChartPathOptions, opts *Options) error {
	if opts.Verify && opts.Keyring == "" {
		return errVerifyWithoutKeyring
	}

	chartPathOptions.Version = opts.Version
	chartPathOptions.RepoURL = opts.RepoURL
	chartPathOptions.Verify = opts.Verify
	chartPathOptions.Keyring = opts.Keyring

	return nil
}
