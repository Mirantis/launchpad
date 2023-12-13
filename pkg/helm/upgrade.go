package helm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/utils/pointer"
)

const (
	defaultUpgradeTimeout = time.Second * 300
)

// ChartDetails contains details about a Helm chart.
type ChartDetails struct {
	// ChartName is the name of the Helm chart.
	ChartName string `yaml:"chartName,omitempty"`
	// ReleaseName is the name of the Helm release.
	ReleaseName string `yaml:"releaseName,omitempty"`
	// RepoURL is the URL to the Helm repository.
	RepoURL string `yaml:"repoURL,omitempty"`
	// Version is the Helm Chart version.
	Version string `yaml:"version,omitempty"`
	// Values contains options for the Helm chart values.
	Values *values.Options `yaml:"values,omitempty"`
	// Installed is true if the chart is installed.
	Installed bool `yaml:"installed,omitempty"`
}

// ChartURL returns a fully qualified chart URL, at this time we only support
// charts obtained from a Helm repository, in the future we may want to support
// local charts either packaged or directories.
func (cd *ChartDetails) ChartURL() (string, error) {
	repoURL, err := url.ParseRequestURI(cd.RepoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Helm repository URL %q: %w", cd.RepoURL, err)
	}

	return fmt.Sprintf("%s/%s", repoURL, cd.ReleaseName), nil
}

// Options to be used with Helm actions.
type Options struct {
	// ChartDetails contains details about the Helm chart.
	ChartDetails
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
	settings := h.settings

	if opts.Timeout == nil {
		opts.Timeout = pointer.Duration(defaultUpgradeTimeout)
	}

	p := getter.All(&settings)

	vals, err := opts.Values.MergeValues(p)
	if err != nil {
		return nil, err
	}

	repoURL, err := opts.ChartDetails.ChartURL()
	if err != nil {
		return nil, err
	}

	ch, err := loader.Load(repoURL)
	if err != nil {
		return nil, err
	}

	// Install the chart if it is not already installed.
	histClient := action.NewHistory(&cfg)
	histClient.Max = 1
	if _, err := histClient.Run(opts.ReleaseName); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			log.Infof("Release %q not found.  Installing it now.", opts.ReleaseName)
			return h.install(ctx, opts, vals, ch)
		}
	}

	log.Infof("Release %q found using chart: %q, upgrading to version: %q", opts.ReleaseName, opts.ChartName, opts.Version)

	u := action.NewUpgrade(&cfg)
	u.Namespace = settings.Namespace()
	u.ReuseValues = opts.ReuseValues
	u.Wait = opts.Wait
	u.Atomic = opts.Atomic
	u.Version = opts.Version
	u.Timeout = *opts.Timeout

	return u.RunWithContext(ctx, opts.ChartDetails.ReleaseName, ch, vals)
}

// install is ran as part of the Upgrade process when the chart is not
// yet installed.  Our Upgrade is essentially the equivalent of
// 'helm upgrade --install'.
func (h *Helm) install(ctx context.Context, opts *Options, vals map[string]interface{}, ch *chart.Chart) (rel *release.Release, err error) {
	cfg := h.config
	settings := h.settings

	i := action.NewInstall(&cfg)

	i.Namespace = settings.Namespace()
	i.Timeout = *opts.Timeout
	i.Version = opts.Version
	i.Atomic = opts.Atomic
	i.Wait = opts.Wait

	return i.RunWithContext(ctx, ch, vals)
}
