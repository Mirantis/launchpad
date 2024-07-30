package kubeclient

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	notFoundErr := &DeploymentNotFoundError{Labels: "app=test"}
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
	notReadyErr := &DeploymentNotReadyError{Labels: "app=test"}
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
	multipleFoundErr := &MultipleDeploymentsFoundError{Labels: "app=test"}
	assert.ErrorAs(t, err, &multipleFoundErr)
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

func TestCreateImagePullSecret(t *testing.T) {
	kc := NewTestClient(t)

	auths := []DockerAuth{
		{
			URL:      "registry.example.com",
			Username: "user",
			Password: "pass",
		},
		{
			URL:      "registry2.example.com",
			Username: "user2",
			Password: "pass2",
		},
	}

	err := kc.CreateImagePullSecret(context.Background(), auths...)
	assert.NoError(t, err)

	// Ensure the secrets were created.
	secret, err := kc.client.CoreV1().Secrets(kc.Namespace).Get(context.Background(), constant.KubernetesDockerRegistryAuthSecretName, metav1.GetOptions{})
	require.NoError(t, err)

	assert.Contains(t, string(secret.Data[".dockerconfigjson"]), `"registry.example.com": {"username": "user", "password": "pass"}`)
	assert.Contains(t, string(secret.Data[".dockerconfigjson"]), `"registry2.example.com": {"username": "user2", "password": "pass2"}`)

	t.Run("Secret already exists", func(t *testing.T) {
		kc := NewTestClient(t)

		auths := []DockerAuth{
			{
				URL:      "registry.example.com",
				Username: "user",
				Password: "pass",
			},
			{
				URL:      "registry3.example.com",
				Username: "user3",
				Password: "pass3",
			},
		}

		// Create a secret with some data already populated.
		_, err = kc.client.CoreV1().Secrets(kc.Namespace).Create(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: constant.KubernetesDockerRegistryAuthSecretName,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths": {"registry.example.com": {"username": "user", "password": "pass"}}}`),
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		err = kc.CreateImagePullSecret(context.Background(), auths...)
		assert.NoError(t, err)

		// Ensure the secrets were created and they contain the new data.
		secret, err := kc.client.CoreV1().Secrets(kc.Namespace).Get(context.Background(), constant.KubernetesDockerRegistryAuthSecretName, metav1.GetOptions{})
		require.NoError(t, err)

		assert.Contains(t, string(secret.Data[".dockerconfigjson"]), `"registry3.example.com": {"username": "user3", "password": "pass3"}`)
	})
}

func TestDeleteSecret(t *testing.T) {
	kc := NewTestClient(t)

	auths := []DockerAuth{
		{
			URL:      "registry.example.com",
			Username: "user",
			Password: "pass",
		},
	}

	err := kc.CreateImagePullSecret(context.Background(), auths...)
	require.NoError(t, err)

	err = kc.DeleteSecret(context.Background(), constant.KubernetesDockerRegistryAuthSecretName)
	assert.NoError(t, err)

	t.Run("The secret does not exist", func(t *testing.T) {
		err = kc.DeleteSecret(context.Background(), constant.KubernetesDockerRegistryAuthSecretName)
		assert.NoError(t, err)
	})
}
