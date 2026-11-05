package helm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartNeedsUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chart upgrade test in short mode (requires OCI chart pull)")
	}
	h := NewHelmTestClient(t)
	rd, _ := InstallCertManagerChart(t, h)

	rdChartVersion := rd.Version

	testCases := []string{
		// ChartNeedsUpgrade returns true if the version is DIFFERENT from
		// the installed one, so anything but Version should return
		// true. It's written this way to support downgrades.
		rdChartVersion,
		"1.0.0",
		"1.0.1",
		"1.0.100",
		"2.1.2",
		"3.0.0-alpha",
		"3.0.0-alpha.1",
		"0.9.9",
		"0.9.0",
		"0.0.1",
		"0.1.1-beta-someothertext",
	}

	for _, tc := range testCases {
		vers, err := version.NewVersion(tc)
		require.NoError(t, err)

		actual, err := h.ChartNeedsUpgrade(rd.ReleaseName, vers)
		if assert.NoError(t, err) {
			if tc == rdChartVersion {
				assert.False(t, actual, "version: %s does match current version: %s", tc, rdChartVersion)
			} else {
				assert.True(t, actual, "version: %s does not match current version: %s", tc, rdChartVersion)
			}
		}
	}
}

// hostileBundleKubeConfig is a kube.yml of the shape a compromised or
// malicious MKE could return in a client bundle: a credential plugin naming a
// local binary of the server's choosing.
const hostileBundleKubeConfig = `apiVersion: v1
kind: Config
clusters:
- name: mke
  cluster:
    server: https://mke.example.invalid:6443
    insecure-skip-tls-verify: true
contexts:
- name: mke
  context:
    cluster: mke
    user: mke
current-context: mke
users:
- name: mke
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: /tmp/launchpad-not-a-real-credential-plugin
      args:
      - --exfiltrate
`

// benignBundleKubeConfig carries credentials but no credential plugin.
const benignBundleKubeConfig = `apiVersion: v1
kind: Config
clusters:
- name: mke
  cluster:
    server: https://mke.example.invalid:6443
    insecure-skip-tls-verify: true
contexts:
- name: mke
  context:
    cluster: mke
    user: mke
current-context: mke
users:
- name: mke
  user:
    token: a-bundle-supplied-token
`

// writeBundleKubeConfig writes contents as the kube.yml of a client bundle
// directory and returns that directory.
func writeBundleKubeConfig(t *testing.T, contents string) string {
	t.Helper()

	bundleDir := t.TempDir()
	err := os.WriteFile(filepath.Join(bundleDir, constant.KubeConfigFile), []byte(contents), 0o600)
	require.NoError(t, err)

	return bundleDir
}

func TestNewFromBundleExecProvider(t *testing.T) {
	t.Run("exec credential plugin is not honoured", func(t *testing.T) {
		h, err := NewFromBundle(writeBundleKubeConfig(t, hostileBundleKubeConfig), "test")
		require.NoError(t, err)

		// Every rest.Config Helm builds comes from this getter, so this is
		// the config the Kubernetes client, the discovery client and the REST
		// mapper are all built from.
		restConfig, err := h.config.RESTClientGetter.ToRESTConfig()
		require.NoError(t, err)

		assert.Nil(t, restConfig.ExecProvider,
			"an exec credential plugin named by the downloaded kubeconfig must not reach the client")
		// Dropping exec must not turn into dropping the connection details.
		assert.Equal(t, "https://mke.example.invalid:6443", restConfig.Host)
	})

	t.Run("credentials without a plugin are left alone", func(t *testing.T) {
		h, err := NewFromBundle(writeBundleKubeConfig(t, benignBundleKubeConfig), "test")
		require.NoError(t, err)

		restConfig, err := h.config.RESTClientGetter.ToRESTConfig()
		require.NoError(t, err)

		assert.Nil(t, restConfig.ExecProvider)
		assert.Equal(t, "a-bundle-supplied-token", restConfig.BearerToken)
	})
}
