//go:build testing

package kubeclient

import (
	"testing"

	msrv1 "github.com/Mirantis/msr-operator/api/v1"
	"github.com/stretchr/testify/require"
	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// NewTestClient returns a new instance of KubeClient for testing purposes, if
// access to the fake clientset is needed, it can be accessed via type assertion
// to *fake.Clientset.
func NewTestClient(t *testing.T) *KubeClient {
	t.Helper()

	namespace := "test"

	client := fake.NewSimpleClientset()
	extendedClient := fakeapiextensions.NewSimpleClientset()

	return &KubeClient{
		Namespace:      namespace,
		client:         client,
		extendedClient: extendedClient,
		config:         nil,
	}
}

// NewTestResourceClient returns a new fake msr.mirantis.com/v1 MSR resource
// client for testing purposes.
func NewTestResourceClient(t *testing.T, namespace string) dynamic.ResourceInterface {
	t.Helper()

	scheme, err := msrv1.SchemeBuilder.Build()
	require.NoError(t, err)

	cl := fakedynamic.NewSimpleDynamicClient(scheme, &msrv1.MSR{})
	return cl.Resource(msrv1.GroupVersion.WithResource("msrs")).Namespace(namespace)
}

// CreateUnstructuredTestMSR returns an unstructured object representing an MSR
// CR for testing.  The name of the MSR is set to "msr-test" and the ready
// status is set to the provided value.
// The fake dynamic.ResourceInterface does not support the use of types
// other than a select few (it panics if the type is not supported), so
// construct a test MSR object that uses these supported types.
// This limitation is only present in the fake client, so this is not an
// issue outside of test.
func CreateUnstructuredTestMSR(t *testing.T, version string, withReadyStatus bool) *unstructured.Unstructured {
	t.Helper()

	msr, err := DecodeIntoUnstructured(&msrv1.MSR{
		ObjectMeta: metav1.ObjectMeta{
			Name: "msr-test",
		},
	})
	require.NoError(t, err)

	msr.Object["spec"] = map[string]interface{}{
		"image": map[string]interface{}{
			"tag": version,
		},
		"nginx": map[string]interface{}{
			"webtls": map[string]interface{}{
				"create": false,
			},
		},
	}

	if withReadyStatus {
		msr.Object["status"] = map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "True",
				},
			},
		}
	}

	return msr
}
