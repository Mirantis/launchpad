package kubeclient

import (
	"context"
	"testing"
	"time"

	msrv1 "github.com/Mirantis/msr-operator/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Mirantis/mcc/pkg/constant"
)

// newTestClient returns a new instance of KubeClient for testing purposes, if
// access to the fake clientset is needed, it can be accessed via type assertion
// to *fake.Clientset.
func newTestClient(t *testing.T) *KubeClient {
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

// newTestResourceClient returns a new fake msr.mirantis.com/v1 MSR resource
// client for testing purposes.
func newTestResourceClient(t *testing.T, namespace string) dynamic.ResourceInterface {
	t.Helper()

	scheme, err := msrv1.SchemeBuilder.Build()
	require.NoError(t, err)

	cl := fakedynamic.NewSimpleDynamicClient(scheme, &msrv1.MSR{})
	return cl.Resource(msrv1.GroupVersion.WithResource("msrs")).Namespace(namespace)
}

func TestCRDReady(t *testing.T) {
	kc := newTestClient(t)

	_, err := kc.extendedClient.ApiextensionsV1().CustomResourceDefinitions().Create(
		context.Background(), &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	err = kc.crdReady(context.Background(), "test")
	assert.NoError(t, err)
}

func TestDeploymentReady(t *testing.T) {
	kc := newTestClient(t)

	err := kc.deploymentReady(context.Background(), "app=test")
	assert.ErrorContains(t, err, "not found")

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas: 1,
		},
	}

	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Create(context.Background(), d, metav1.CreateOptions{})
	require.NoError(t, err)

	err = kc.deploymentReady(context.Background(), "app=test")
	assert.ErrorContains(t, err, "found", "not yet ready")

	d.Status.ReadyReplicas = 1

	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Update(context.Background(), d, metav1.UpdateOptions{})
	require.NoError(t, err)

	err = kc.deploymentReady(context.Background(), "app=test")
	assert.NoError(t, err)

	d.ObjectMeta.Name = "test2"
	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Create(context.Background(), d, metav1.CreateOptions{})
	require.NoError(t, err)

	err = kc.deploymentReady(context.Background(), "app=test")
	assert.ErrorContains(t, err, "found more than once")
}

func TestDecodeIntoUnstructured(t *testing.T) {
	msr := &msrv1.MSR{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "msr.mirantis.com/v1",
			Kind:       "MSR",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "msr-test",
		},
		Spec: msrv1.MSRSpec{
			License: "veryrealisticlicense",
		},
		Status: msrv1.MSRStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	obj, err := DecodeIntoUnstructured(msr)
	require.NoError(t, err)

	assert.Equal(t, "msr-test", obj.GetName())
	assert.Equal(t, "msr.mirantis.com/v1", obj.GetAPIVersion())
	assert.Equal(t, "MSR", obj.GetKind())
	assert.Equal(t, "veryrealisticlicense", obj.Object["spec"].(map[string]interface{})["license"])
	assert.NotZero(t, obj.GetCreationTimestamp())

	t.Run("Given object's creation timestamp is not stripped if present", func(t *testing.T) {
		msr.ObjectMeta.CreationTimestamp = metav1.Now()
		obj, err = DecodeIntoUnstructured(msr)
		require.NoError(t, err)
		if assert.NotZero(t, obj.GetCreationTimestamp()) {
			time.Sleep(1 * time.Second)
			// obj.GetCreationTimestamp() returns local time in RFC3339 format.
			assert.Equal(t, msr.ObjectMeta.CreationTimestamp.Rfc3339Copy(), obj.GetCreationTimestamp())
		}
	})
}

// createUnstructuredTestMSR returns an unstructured object representing an MSR
// CR for testing.  The name of the MSR is set to "msr-test" and the ready
// status is set to the provided value.
// The fake dynamic.ResourceInterface does not support the use of types
// other than a select few (it panics if the type is not supported), so
// construct a test MSR object that uses these supported types.
// This limitation is only present in the fake client, so this is not an
// issue outside of test.
func createUnstructuredTestMSR(t *testing.T, withReadyStatus bool) *unstructured.Unstructured {
	t.Helper()

	msr, err := DecodeIntoUnstructured(&msrv1.MSR{
		ObjectMeta: metav1.ObjectMeta{
			Name: "msr-test",
		},
	})
	require.NoError(t, err)

	msr.Object["spec"] = map[string]interface{}{
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

func TestCRIsReady(t *testing.T) {
	kc := newTestClient(t)

	rc := newTestResourceClient(t, kc.Namespace)

	msr := createUnstructuredTestMSR(t, false)

	_, err := kc.crIsReady(context.Background(), msr, rc)
	assert.ErrorContains(t, err, "not found")

	_, err = rc.Create(context.Background(), msr, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = kc.crIsReady(context.Background(), msr, rc)
	assert.ErrorContains(t, err, "status.conditions not found")

	msr = createUnstructuredTestMSR(t, true)

	_, err = rc.Update(context.Background(), msr, metav1.UpdateOptions{})
	require.NoError(t, err)

	_, err = kc.crIsReady(context.Background(), msr, rc)
	assert.NoError(t, err)
}

func TestSetStorageClassDefault(t *testing.T) {
	kc := newTestClient(t)

	logrus.SetLevel(logrus.DebugLevel)

	_, err := kc.client.StorageV1().StorageClasses().Create(context.Background(), &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-sc-that-exists",
			Annotations: map[string]string{
				constant.DefaultStorageClassAnnotation: "true",
			},
		}}, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("Given StorageClass to set to default does not exist", func(t *testing.T) {
		err := kc.SetStorageClassDefault(context.Background(), "test-sc")
		assert.ErrorContains(t, err, "not found", "test-sc")
	})

	t.Run("Default storage class exists", func(t *testing.T) {
		// Create the StorageClass we intend to set as default.
		_, err = kc.client.StorageV1().StorageClasses().Create(context.Background(), &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-sc",
			}}, metav1.CreateOptions{})
		require.NoError(t, err)

		err = kc.SetStorageClassDefault(context.Background(), "test-sc")
		assert.NoError(t, err)

		// Ensure the old default StorageClass no longer has the default
		// annotation.
		sc, err := kc.client.StorageV1().StorageClasses().Get(context.Background(), "some-sc-that-exists", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Empty(t, sc.Annotations[constant.DefaultStorageClassAnnotation])

		// Ensure the new default StorageClass has the default annotation.
		newSC, err := kc.client.StorageV1().StorageClasses().Get(context.Background(), "test-sc", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "true", newSC.Annotations[constant.DefaultStorageClassAnnotation])
	})

}
