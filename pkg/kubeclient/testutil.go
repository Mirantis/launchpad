//go:build testing

package kubeclient

import (
	"testing"

	msrv1 "github.com/Mirantis/msr-operator/api/v1"
	"github.com/stretchr/testify/require"
	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
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
