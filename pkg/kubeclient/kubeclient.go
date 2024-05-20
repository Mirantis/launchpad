package kubeclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/constant"
	log "github.com/sirupsen/logrus"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	Namespace string

	client         kubernetes.Interface
	extendedClient apiextensionsclientset.Interface
	config         *rest.Config
}

// NewFromBundle returns a new instance of KubeClient from a
// given bundle directory and defaulting to the provided namespace.
func NewFromBundle(bundleDir, namespace string) (*KubeClient, error) {
	f := filepath.Join(bundleDir, constant.KubeConfigFile)

	configBytes, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", f, err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	return New(config, namespace)
}

// New creates a new instance of KubeClient from a given namespace and
// *rest.Config.
func New(config *rest.Config, namespace string) (*KubeClient, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not initialize kubernetes client: %w", err)
	}

	extendedClientSet, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize apiextensions clientset: %w", err)
	}

	return &KubeClient{
		Namespace:      namespace,
		client:         clientSet,
		extendedClient: extendedClientSet,
		config:         config,
	}, nil
}

type NotFoundError struct {
	Kind string
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s: %q not found", e.Kind, e.Name)
}

// crdReady verifies that the CRD is available.
// This is the equivalent of running `kubectl get crd crdName`.
func (kc *KubeClient) crdReady(ctx context.Context, name string) error {
	_, err := kc.extendedClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get %q crd: %w", name, err)
	}

	return nil
}

type DeploymentNotReadyError struct {
	Labels string
}

func (e *DeploymentNotReadyError) Error() string {
	return fmt.Sprintf("deployment with %q labels was found, but is not yet ready", e.Labels)
}

type DeploymentNotFoundError struct {
	Labels string
}

func (e *DeploymentNotFoundError) Error() string {
	return fmt.Sprintf("deployment with %q labels not found", e.Labels)
}

type MultipleDeploymentsFoundError struct {
	Labels string
}

func (e *MultipleDeploymentsFoundError) Error() string {
	return fmt.Sprintf("deployment with %q labels found more than once, ensure the deployment is unique", e.Labels)
}

// deploymentReady verifies that the Deployment is available.
// This is the equivalent of running `kubectl list deployment -l labels`.
// The labels should be formatted as a comma separated list of key=value pairs.
func (kc *KubeClient) deploymentReady(ctx context.Context, labels string) error {
	deployments, err := kc.client.AppsV1().Deployments(kc.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &DeploymentNotFoundError{Labels: labels}
		}

		return fmt.Errorf("failed to list deployments with labels: %q: %w", labels, err)
	}

	if len(deployments.Items) > 1 {
		return &MultipleDeploymentsFoundError{Labels: labels}
	}

	if len(deployments.Items) < 1 {
		return &DeploymentNotFoundError{Labels: labels}
	}

	if deployments.Items[0].Status.ReadyReplicas < 1 {
		return &DeploymentNotReadyError{Labels: labels}
	}

	return nil
}

type ConditionsNotFoundError struct {
	Kind string
	Name string
}

func (e *ConditionsNotFoundError) Error() string {
	return fmt.Sprintf("status.conditions not found in %s: %q", e.Kind, e.Name)
}

// crIsReady verifies that the Custom Resource is available and ready, the CR
// object to check for should be provided as an unstructured object, a
// resourceClient affiliated with the CR is also required.
func (kc *KubeClient) crIsReady(ctx context.Context, obj *unstructured.Unstructured, resourceClient dynamic.ResourceInterface) (bool, error) {
	name := obj.GetName()

	crdObj, err := resourceClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get resource: %q: %w", name, err)
	}

	// Check readiness condition
	conditions, found, err := unstructured.NestedSlice(crdObj.Object, "status", "conditions")
	if err != nil || !found {
		if !found {
			return false, &ConditionsNotFoundError{Kind: obj.GetKind(), Name: name}
		}

		return false, fmt.Errorf("failed to get status.conditions from CR: %q: %w", name, err)
	}

	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
			if readyStatus, ok := condMap["status"].(string); ok && readyStatus == "True" {
				return true, nil
			}
		}
	}

	return false, nil
}

type StorageClassNotFoundError struct {
	Name string
}

func (e *StorageClassNotFoundError) Error() string {
	return fmt.Sprintf("storage class: %q not found", e.Name)
}

// SetStorageClassDefault configures the given StorageClass name as the default,
// ensuring that no other StorageClass has the default annotation.
func (kc *KubeClient) SetStorageClassDefault(ctx context.Context, name string) error {
	log.Debugf("setting: %s as default StorageClass", name)

	storageClasses, err := kc.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list StorageClasses: %w", err)
	}

	var needsUpdate, found bool

	for _, storageClass := range storageClasses.Items {
		storageClass := storageClass

		if storageClass.Name == name {
			found = true

			// Apply the default annotation to the named StorageClass.
			if _, ok := storageClass.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation]; !ok {
				log.Debugf("setting default StorageClass annotation on: %s", name)

				if storageClass.ObjectMeta.Annotations == nil {
					storageClass.ObjectMeta.Annotations = make(map[string]string)
				}

				storageClass.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation] = "true"
			}
			needsUpdate = true
		} else {
			// Strip the default annotation if found from a different StorageClass.
			if _, ok := storageClass.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation]; ok {
				log.Debugf("found existing default StorageClass: %s, removing default annotation", storageClass.Name)

				delete(storageClass.ObjectMeta.Annotations, constant.DefaultStorageClassAnnotation)
				needsUpdate = true
			}
		}

		if needsUpdate {
			if _, err := kc.client.StorageV1().StorageClasses().Update(ctx, &storageClass, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update StorageClass: %q annotations: %w", name, err)
			}
		}
	}

	// If the named StorageClass was not found, return an error.
	if !found {
		return &StorageClassNotFoundError{Name: name}
	}

	return nil
}

// DeleteService deletes service by name.
func (kc *KubeClient) DeleteService(ctx context.Context, name string) error {
	if err := kc.client.CoreV1().Services(kc.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete service: %q: %w", name, err)
	}

	return nil
}

var errNotYetImplemented = errors.New("not yet implemented")

// ExposeLoadBalancer creates a new service of Type: LoadBalancer, it's
// the equivalent of 'kubectl expose'.
func (kc *KubeClient) ExposeLoadBalancer(_ context.Context, _ string) error {
	return errNotYetImplemented
}

// DecodeIntoUnstructured converts a given runtime.Object into an
// unstructured object for use with a dynamic client.
func DecodeIntoUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	return &unstructured.Unstructured{Object: result}, nil
}
