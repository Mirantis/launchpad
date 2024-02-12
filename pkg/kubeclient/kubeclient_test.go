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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Mirantis/mcc/pkg/constant"
)

func TestCRDReady(t *testing.T) {
	kc := NewTestClient(t)

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
	kc := NewTestClient(t)

	err := kc.deploymentReady(context.Background(), "app=test")
	notFoundErr := &ErrDeploymentNotFound{Labels: "app=test"}
	assert.ErrorAs(t, err, &notFoundErr)

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
	notReadyErr := &ErrDeploymentNotReady{Labels: "app=test"}
	assert.ErrorAs(t, err, &notReadyErr)

	d.Status.ReadyReplicas = 1

	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Update(context.Background(), d, metav1.UpdateOptions{})
	require.NoError(t, err)

	err = kc.deploymentReady(context.Background(), "app=test")
	assert.NoError(t, err)

	d.ObjectMeta.Name = "test2"
	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Create(context.Background(), d, metav1.CreateOptions{})
	require.NoError(t, err)

	err = kc.deploymentReady(context.Background(), "app=test")
	multipleFoundErr := &ErrMultipleDeploymentsFound{Labels: "app=test"}
	assert.ErrorAs(t, err, &multipleFoundErr)
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

	t.Run("Given object's creation timestamp is not stripped if present", func(t *testing.T) {
		// Set the object to have been created 10 seconds ago.
		msr.ObjectMeta.CreationTimestamp = metav1.NewTime(time.Now().Add(-10 * time.Second))
		obj, err = DecodeIntoUnstructured(msr)
		require.NoError(t, err)
		if assert.NotZero(t, obj.GetCreationTimestamp()) {
			// obj.GetCreationTimestamp() returns local time in RFC3339 format.
			assert.Equal(t, metav1.Time{Time: msr.ObjectMeta.CreationTimestamp.Rfc3339Copy().Local()}, obj.GetCreationTimestamp())
		}
	})
}

func TestCRIsReady(t *testing.T) {
	kc := NewTestClient(t)
	rc := NewTestResourceClient(t, kc.Namespace)

	msr := CreateUnstructuredTestMSR(t, "1.0.0", false)

	_, err := kc.crIsReady(context.Background(), msr, rc)
	assert.ErrorContains(t, err, "not found")

	_, err = rc.Create(context.Background(), msr, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = kc.crIsReady(context.Background(), msr, rc)
	assert.ErrorContains(t, err, "status.conditions not found")

	msr = CreateUnstructuredTestMSR(t, "1.0.0", true)

	_, err = rc.Update(context.Background(), msr, metav1.UpdateOptions{})
	require.NoError(t, err)

	_, err = kc.crIsReady(context.Background(), msr, rc)
	assert.NoError(t, err)
}

func TestSetStorageClassDefault(t *testing.T) {
	kc := NewTestClient(t)

	initialLogLevel := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(initialLogLevel) })
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
		if assert.NoError(t, err) {
			// Ensure the old default StorageClass no longer has the default
			// annotation.
			sc, err := kc.client.StorageV1().StorageClasses().Get(context.Background(), "some-sc-that-exists", metav1.GetOptions{})
			if assert.NoError(t, err) {
				assert.Empty(t, sc.Annotations[constant.DefaultStorageClassAnnotation])
			}
			// Ensure the new default StorageClass has the default annotation.
			newSC, err := kc.client.StorageV1().StorageClasses().Get(context.Background(), "test-sc", metav1.GetOptions{})
			if assert.NoError(t, err) {
				assert.Equal(t, "true", newSC.Annotations[constant.DefaultStorageClassAnnotation])
			}
		}
	})

}
