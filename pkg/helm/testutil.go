// go:build testing

package helm

import (
	"testing"

	"github.com/Mirantis/mcc/pkg/kubeclient"
)

func NewHelmTestClient(t *testing.T, kc *kubeclient.KubeClient) *Helm {
	t.Helper()

	return &Helm{}
}
