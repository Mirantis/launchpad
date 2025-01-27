package kubeclient

import (
	"context"
	"fmt"
	"testing"

	github.com/Mirantis/launchpad/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	u "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateMSROperatorReady(t *testing.T) {
	kc := NewTestClient(t)

	err := kc.ValidateMSROperatorReady(context.Background())
	assert.ErrorContains(t, err, "not found")

	_, err = kc.extendedClient.ApiextensionsV1().CustomResourceDefinitions().Create(
		context.Background(), &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "msrs.msr.mirantis.com",
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	msrOperatorDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "msr-operator",
			Labels: map[string]string{
				"app.kubernetes.io/name": "msr-operator",
			},
		},
	}

	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Create(
		context.Background(), msrOperatorDeployment, metav1.CreateOptions{})
	require.NoError(t, err)

	err = kc.ValidateMSROperatorReady(context.Background())
	assert.ErrorContains(t, err, "found", "not yet ready")

	msrOperatorDeployment.Status.ReadyReplicas = 1
	_, err = kc.client.AppsV1().Deployments(kc.Namespace).Update(
		context.Background(), msrOperatorDeployment, metav1.UpdateOptions{})
	require.NoError(t, err)

	err = kc.ValidateMSROperatorReady(context.Background())
	assert.NoError(t, err)
}

func TestApplyMSRCR(t *testing.T) {
	kc := NewTestClient(t)
	rc := NewTestResourceClient(t, kc.Namespace)

	msr := CreateUnstructuredTestMSR(t, "1.0.0", true)

	t.Run("MSR CR does not yet exist", func(t *testing.T) {
		t.Cleanup(func() {
			err := kc.DeleteMSRCR(context.Background(), "msr-test", rc)
			require.NoError(t, err)
		})

		err := kc.ApplyMSRCR(context.Background(), msr, rc)
		assert.NoError(t, err)

		actual, err := kc.GetMSRCR(context.Background(), "msr-test", rc)
		require.NoError(t, err)

		assert.Equal(t, "msr-test", actual.GetName())
	})

	t.Run("MSR CR already exists and is updated", func(t *testing.T) {
		newMsr := CreateUnstructuredTestMSR(t, "1.0.0", true)

		// Update the license within the unstructured object.
		newMsr.Object["spec"].(map[string]interface{})["license"] = "areallicense"

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		err = kc.ApplyMSRCR(context.Background(), newMsr, rc)
		assert.NoError(t, err)

		actual, err := kc.GetMSRCR(context.Background(), "msr-test", rc)
		require.NoError(t, err)

		assert.Equal(t, "msr-test", actual.GetName())

		actualLicense, _, err := u.NestedString(actual.Object, "spec", "license")
		require.NoError(t, err)

		assert.Equal(t, "areallicense", actualLicense)
	})
}

func TestPrepareNodeForMSR(t *testing.T) {
	kc := NewTestClient(t)

	kc.client.CoreV1().Nodes().Create(context.Background(), &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{
					Key:   constant.KubernetesOrchestratorTaint,
					Value: "NoExecute",
				},
			},
		},
	}, metav1.CreateOptions{})

	err := kc.PrepareNodeForMSR(context.Background(), "node1")
	assert.NoError(t, err)

	actualNode, err := kc.client.CoreV1().Nodes().Get(context.Background(), "node1", metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "true", actualNode.Labels[constant.MSRNodeSelector])
	assert.Empty(t, actualNode.Spec.Taints)
}

func TestMSRURL(t *testing.T) {
	t.Run("no spec.service.externalHTTPSPort", func(t *testing.T) {
		kc := NewTestClient(t)
		rc := NewTestResourceClient(t, kc.Namespace)

		msr := CreateUnstructuredTestMSR(t, "1.0.0", true)

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = kc.MSRURL(context.Background(), "msr-test", rc)
		assert.ErrorContains(t, err, "spec.service.externalHTTPSPort", "not found")
	})

	t.Run("no msr-test service found", func(t *testing.T) {
		kc := NewTestClient(t)
		rc := NewTestResourceClient(t, kc.Namespace)

		msr := CreateUnstructuredTestMSR(t, "1.0.0", true)

		serviceTypes := []string{
			"NodePort",
			"LoadBalancer",
			"ClusterIP",
		}

		for _, st := range serviceTypes {
			msr.Object["spec"] = map[string]interface{}{
				"service": map[string]interface{}{
					"externalHTTPSPort": int64(4443),
					"serviceType":       st,
				},
			}
		}

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = kc.MSRURL(context.Background(), "msr-test", rc)
		assert.ErrorContains(t, err, "failed to get service", "msr-test")
	})

	t.Run("no MSR nodes found, serviceType: NodePort", func(t *testing.T) {
		kc := NewTestClient(t)
		rc := NewTestResourceClient(t, kc.Namespace)

		msr := CreateUnstructuredTestMSR(t, "1.0.0", true)

		msr.Object["spec"] = map[string]interface{}{
			"service": map[string]interface{}{
				"externalHTTPSPort": int64(4443),
				"serviceType":       "NodePort",
			},
		}

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		createTestService(t, kc, "NodePort", 4443, false)

		_, err = kc.MSRURL(context.Background(), "msr-test", rc)

		notFoundErr := &NotFoundError{}
		assert.ErrorAs(t, err, &notFoundErr)
	})

	t.Run("unsupported spec.serviceType", func(t *testing.T) {
		kc := NewTestClient(t)
		rc := NewTestResourceClient(t, kc.Namespace)

		msr := CreateUnstructuredTestMSR(t, "1.0.0", true)

		msr.Object["spec"] = map[string]interface{}{
			"service": map[string]interface{}{
				"externalHTTPSPort": int64(4443),
				"serviceType":       "SomeServiceType",
			},
		}

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		createTestService(t, kc, "SomeServiceType", 4443, false)

		_, err = kc.MSRURL(context.Background(), "msr-test", rc)

		unknownErr := &UnknownServiceTypeError{ServiceType: "SomeServiceType"}
		assert.ErrorAs(t, err, &unknownErr)
	})

	testCases := []struct {
		externalPort int64
		serviceType  string
		headless     bool
		expected     string
	}{
		// NodePort should print the node port within the service spec instead
		// of the external port.
		{externalPort: 4443, serviceType: "NodePort", headless: false, expected: "https://msr.example.com:30001"},
		{externalPort: 4443, serviceType: "ClusterIP", headless: true, expected: "https://msr-nginx.test.svc.cluster.local:4443"},
		{externalPort: 443, serviceType: "ClusterIP", headless: true, expected: "https://msr-nginx.test.svc.cluster.local"},
		{externalPort: 4443, serviceType: "ClusterIP", headless: false, expected: "https://10.11.12.13:4443"},
		{externalPort: 443, serviceType: "ClusterIP", headless: false, expected: "https://10.11.12.13"},
		{externalPort: 443, serviceType: "LoadBalancer", headless: false, expected: "https://10.12.14.16"},
		{externalPort: 4443, serviceType: "LoadBalancer", headless: false, expected: "https://10.12.14.16:4443"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("serviceType: %s, headless: %t, externalPort: %d", tc.serviceType, tc.headless, tc.externalPort), func(t *testing.T) {
			tc := tc

			kc := NewTestClient(t)
			rc := NewTestResourceClient(t, kc.Namespace)

			if tc.headless {
				// An nginx pod needs to exist to construct service details.
				_, err := kc.client.CoreV1().Pods(kc.Namespace).Create(context.Background(), &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "msr-nginx",
						Labels: map[string]string{
							"app.kubernetes.io/name":      "msr-test",
							"app.kubernetes.io/instance":  "msr-test",
							"app.kubernetes.io/component": "nginx",
						},
					},
				}, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			kc.client.CoreV1().Nodes().Create(context.Background(), &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "msr-node1",
					Labels: map[string]string{
						constant.MSRNodeSelector: "true",
					},
				},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeExternalDNS,
							Address: "msr.example.com",
						},
						{
							Type:    corev1.NodeExternalIP,
							Address: "123.13.41.15",
						},
					},
				},
			}, metav1.CreateOptions{})

			msrCR := CreateUnstructuredTestMSR(t, "1.0.0", true)

			msrCR.Object["spec"] = map[string]interface{}{
				"service": map[string]interface{}{
					"externalHTTPSPort": tc.externalPort,
					"serviceType":       tc.serviceType,
				},
			}

			_, err := rc.Create(context.Background(), msrCR, metav1.CreateOptions{})
			require.NoError(t, err)

			createTestService(t, kc, tc.serviceType, int(tc.externalPort), tc.headless)

			url, err := kc.MSRURL(context.Background(), "msr-test", rc)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, url.String())
		})
	}
}

func createTestService(t *testing.T, kc *KubeClient, serviceType string, externalPort int, headless bool) {
	t.Helper()

	_, err := kc.client.CoreV1().Services(kc.Namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "msr-test",
		},
		Spec: func() corev1.ServiceSpec {
			spec := corev1.ServiceSpec{
				Type: corev1.ServiceType(serviceType),
				Ports: []corev1.ServicePort{
					{
						Port: int32(externalPort),
					},
				},
			}

			if serviceType == "NodePort" {
				spec.Ports[0].NodePort = 30001
			}

			if serviceType == "ClusterIP" {
				if headless {
					spec.ClusterIP = "None"
				} else {
					spec.ClusterIP = "10.11.12.13"
				}
			}

			return spec
		}(),
		Status: func() corev1.ServiceStatus {
			if serviceType == "LoadBalancer" {
				return corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP:       "10.12.14.16",
								Hostname: "msr.example.com",
							},
						},
					},
				}
			}

			return corev1.ServiceStatus{}
		}(),
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}
