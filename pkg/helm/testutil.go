// go:build testing

package helm

import (
	"context"
	"io"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/require"

	"github.com/Mirantis/mcc/pkg/constant"
)

type HelmTestClientOption func(*helmTestClientOptions)

type helmTestClientOptions struct {
	failing *kubefake.FailingKubeClient
}

// WithFailingKubeClient configures the Helm test client to use a failing
// kube client for testing failing scenarios.
func WithFailingKubeClient(failing *kubefake.FailingKubeClient) HelmTestClientOption {
	return func(o *helmTestClientOptions) {
		o.failing = failing
	}
}

func gatherOptions(options []HelmTestClientOption) *helmTestClientOptions {
	o := &helmTestClientOptions{}

	for _, opt := range options {
		opt(o)
	}

	return o
}

// NewHelmTestClient creates a new Helm for testing purposes and returns a
// writer to capture the output of the Helm client.  Pass a
// *kubefake.FailingKubeClient with options to simulate failing Helm actions,
// for a passing Helm client pass nil.
func NewHelmTestClient(t *testing.T, options ...HelmTestClientOption) *Helm {
	t.Helper()

	opts := gatherOptions(options)

	registryClient, err := registry.NewClient()
	require.NoError(t, err)

	config := action.Configuration{
		Releases: storage.Init(driver.NewMemory()),
		KubeClient: func() kube.Interface {
			if opts.failing != nil {
				return opts.failing
			}

			return &kubefake.PrintingKubeClient{Out: io.Discard}
		}(),
		Capabilities:   chartutil.DefaultCapabilities,
		RegistryClient: registryClient,
		Log:            t.Logf,
	}

	settings := cli.New()
	settings.SetNamespace("test")

	return &Helm{
		config:   config,
		settings: *settings,
	}
}

// InstallRethinkDBOperatorChart installs version 1.0.0 of rethinkdb-operator
// to use as a chart to query for testing purposes and returns the
// ReleaseDetails for the chart as well as a function to uninstall the chart.
func InstallRethinkDBOperatorChart(t *testing.T, h *Helm) (ReleaseDetails, func()) {
	t.Helper()

	rd := ReleaseDetails{
		ChartName:   constant.RethinkDBOperator,
		ReleaseName: constant.RethinkDBOperator,
		RepoURL:     "https://registry.mirantis.com/charts/rethinkdb/rethinkdb-operator",
		Version:     "1.0.0",
	}

	_, err := h.Upgrade(context.Background(), &Options{
		ReleaseDetails: rd, Timeout: ptr.To(DefaultTimeout),
	})
	require.NoError(t, err)

	uninstallFunc := func() {
		err := h.Uninstall(context.Background(), &Options{
			ReleaseDetails: rd, Timeout: ptr.To(DefaultTimeout),
		})
		require.NoError(t, err)
	}

	rd.Installed = true

	return rd, uninstallFunc
}
