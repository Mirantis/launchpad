package kubeclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/mapstructure"
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

	"github.com/Mirantis/mcc/pkg/constant"
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

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not initialize kubernetes client: %w", err)
	}

	extendedClientSet, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize apiextensions clientset: %w", err)
	}

	kc := &KubeClient{
		Namespace:      namespace,
		client:         clientSet,
		extendedClient: extendedClientSet,
		config:         config,
	}

	return kc, nil
}

// CRDReady verifies that the CRD is available.
// This is the equivalent of running `kubectl get crd crdName`.
func (kc *KubeClient) crdReady(ctx context.Context, name string) error {
	_, err := kc.extendedClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get %q crd: %w", name, err)
	}

	return nil
}

// deploymentReady verifies that the Deployment is available.
// This is the equivalent of running `kubectl list deployment -l labels`.
// The labels should be formatted as a comma separated list of key=value pairs.
func (kc *KubeClient) deploymentReady(ctx context.Context, labels string) error {
	d, err := kc.client.AppsV1().Deployments(kc.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("deployment with %q labels not found, ensure the deployment exists", labels)
		}

		return err
	}

	if len(d.Items) > 1 {
		return fmt.Errorf("deployment with %q labels found more than once, ensure the deployment is unique", labels)
	}

	if len(d.Items) < 1 {
		return fmt.Errorf("deployment with %q labels not found, ensure the deployment exists", labels)
	}

	if d.Items[0].Status.ReadyReplicas < 1 {
		return fmt.Errorf("deployment with %q labels was found, but is not yet ready", labels)
	}

	return nil
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
			return false, fmt.Errorf("status.conditions not found in CR: %q", name)
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

// SetStorageClassDefault configures the given StorageClass name as the default,
// ensuring that no other StorageClass has the default annotation.
func (kc *KubeClient) SetStorageClassDefault(ctx context.Context, name string) error {
	log.Debugf("setting: %s as default StorageClass", name)

	storageClasses, err := kc.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list StorageClasses: %w", err)
	}

	var needsUpdate, found bool

	for _, sc := range storageClasses.Items {
		if sc.Name == name {
			found = true

			// Apply the default annotation to the named StorageClass.
			if _, ok := sc.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation]; !ok {
				log.Debugf("setting default StorageClass annotation on: %s", name)

				if sc.ObjectMeta.Annotations == nil {
					sc.ObjectMeta.Annotations = make(map[string]string)
				}

				sc.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation] = "true"
			}
			needsUpdate = true
		} else {
			// Strip the default annotation if found from a different StorageClass.
			if _, ok := sc.ObjectMeta.Annotations[constant.DefaultStorageClassAnnotation]; ok {
				log.Debugf("found existing default StorageClass: %s, removing default annotation", sc.Name)

				delete(sc.ObjectMeta.Annotations, constant.DefaultStorageClassAnnotation)
				needsUpdate = true
			}
		}

		if needsUpdate {
			// Apply the annotation modifications if they were made.
			if _, err := kc.client.StorageV1().StorageClasses().Update(ctx, &sc, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update StorageClass: %q annotations: %w", name, err)
			}
		}
	}

	// If the named StorageClass was not found, return an error.
	if !found {
		return fmt.Errorf("StorageClass: %q not found", name)
	}

	return nil
}

// DeleteService deletes service by name.
func (kc *KubeClient) DeleteService(ctx context.Context, name string) error {
	return kc.client.CoreV1().Services(kc.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ExposeLoadBalancer creates a new service of Type: LoadBalancer, it's
// the equivalent of 'kubectl expose'.
func (kc *KubeClient) ExposeLoadBalancer(ctx context.Context, url string) error {
	return fmt.Errorf("not yet implemented")
}

// DecodeIntoUnstructured converts a given runtime.Object into an
// unstructured object for use with a dynamic client.
func DecodeIntoUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	result := make(map[string]interface{})

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &result,
		TagName: "json",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := d.Decode(obj); err != nil {
		return nil, fmt.Errorf("failed to decode object into map: %w", err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: result}

	// Set specific fields to ensure the object is valid and remove the TypeMeta
	// field, this is to workaround an issue with mapstructure decoding "inline"
	// tagged fields into map, we're effectively rebuilding the inlined TypeMeta
	// fields here.
	unstructuredObj.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	if unstructuredObj.GetCreationTimestamp().Time.IsZero() {
		unstructuredObj.SetCreationTimestamp(metav1.NewTime(time.Now().UTC()))
	}

	return &unstructured.Unstructured{Object: result}, nil
}
