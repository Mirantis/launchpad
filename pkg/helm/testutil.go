// go:build testing

package helm

import (
	"context"
	"io"
	"testing"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/utils/ptr"
)

type TestClientOption func(*testClientOptions)

type testClientOptions struct {
	failing *kubefake.FailingKubeClient
}

// WithFailingKubeClient configures the Helm test client to use a failing
// kube client for testing failing scenarios.
func WithFailingKubeClient(failing *kubefake.FailingKubeClient) TestClientOption {
	return func(o *testClientOptions) {
		o.failing = failing
	}
}

func gatherOptions(options []TestClientOption) *testClientOptions {
	o := &testClientOptions{}

	for _, opt := range options {
		opt(o)
	}

	return o
}

// NewHelmTestClient creates a new Helm for testing purposes. Pass a
// *kubefake.FailingKubeClient with options to simulate failing Helm actions.
func NewHelmTestClient(t *testing.T, options ...TestClientOption) *Helm {
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

// InstallRethinkDBOperatorChart installs rethinkdb-operator
// to use as a chart to query for testing purposes and returns the
// ReleaseDetails for the chart as well as a function to uninstall the chart.
func InstallRethinkDBOperatorChart(t *testing.T, helmClient *Helm) (ReleaseDetails, func()) {
	t.Helper()

	releaseDetails := ReleaseDetails{
		ChartName:   constant.RethinkDBOperator,
		ReleaseName: constant.RethinkDBOperator,
		RepoURL:     "https://registry.mirantis.com/charts/rethinkdb/rethinkdb-operator",
		Version:     "1.0.0",
	}

	_, err := helmClient.Upgrade(context.Background(), &Options{
		ReleaseDetails: releaseDetails, Timeout: ptr.To(DefaultTimeout),
	})
	require.NoError(t, err)

	uninstallFunc := func() {
		err := helmClient.Uninstall(&Options{
			ReleaseDetails: releaseDetails, Timeout: ptr.To(DefaultTimeout),
		})
		require.NoError(t, err)
	}

	releaseDetails.Installed = true

	return releaseDetails, uninstallFunc
}
