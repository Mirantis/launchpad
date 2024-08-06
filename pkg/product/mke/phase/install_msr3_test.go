package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/Mirantis/mcc/pkg/constant"
)

func TestAppendImagePullSecret(t *testing.T) {
	t.Run("spec.image.pullSecret is populated", func(t *testing.T) {
		testSecretAppended(t, &unstructured.Unstructured{
			Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"image": map[string]interface{}{
						"pullSecret": []interface{}{
							map[string]interface{}{
								"name": "my-secret",
							},
						},
					},
				},
			},
		}, "my-secret", constant.KubernetesDockerRegistryAuthSecretName)
	})

	t.Run("spec.image.pullSecret is not populated", func(t *testing.T) {
		testSecretAppended(t, &unstructured.Unstructured{Object: map[string]interface{}{}}, constant.KubernetesDockerRegistryAuthSecretName)
	})
}

func testSecretAppended(t *testing.T, unstrObj *unstructured.Unstructured, expectedSecretNames ...string) {
	t.Helper()

	err := appendImagePullSecret(unstrObj, "spec", "image", "pullSecret")
	require.NoError(t, err)

	actualSecrets, found, err := unstructured.NestedSlice(unstrObj.Object, "spec", "image", "pullSecret")
	require.True(t, found)
	require.NoError(t, err)

	assert.Len(t, actualSecrets, len(expectedSecretNames))
	for _, secretName := range expectedSecretNames {
		assert.Contains(t, actualSecrets, map[string]interface{}{"name": secretName})
	}

	// Ensure that the unstructured object can be marshaled into valid YAML.
	_, err = yaml.Marshal(unstrObj)
	assert.NoError(t, err)
}
