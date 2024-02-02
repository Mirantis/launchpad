// go:build testing

package helm

import (
	"bufio"
	"bytes"
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

// NewHelmTestClient creates a new Helm for testing purposes and returns a
// writer to capture the output of the Helm client.  Pass a
// *kubefake.FailingKubeClient with options to simulate failing Helm actions,
// for a passing Helm client pass nil.
func NewHelmTestClient(t *testing.T, failing *kubefake.FailingKubeClient) (*Helm, io.Writer) {
	t.Helper()

	registryClient, err := registry.NewClient()
	require.NoError(t, err)

	var b bytes.Buffer
	out := bufio.NewWriter(&b)

	config := action.Configuration{
		Releases: storage.Init(driver.NewMemory()),
		KubeClient: func() kube.Interface {
			printer := &kubefake.PrintingKubeClient{Out: out}

			if failing == nil {
				return printer
			}

			failing.PrintingKubeClient = *printer

			return failing
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
	}, out
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
