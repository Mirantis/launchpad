package kubeclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util/pollutil"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func (kc *KubeClient) GetMSRCR(ctx context.Context, name string, rc dynamic.ResourceInterface) (*unstructured.Unstructured, error) {
	unstructured, err := rc.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR CR: %w", err)
	}

	return unstructured, nil
}

// DeleteMSRCR is a wrapper around resourceClient.Delete for the MSR resource.
func (kc *KubeClient) DeleteMSRCR(ctx context.Context, name string, rc dynamic.ResourceInterface) error {
	if err := rc.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete MSR CR: %w", err)
	}

	return nil
}

func (kc *KubeClient) ValidateMSROperatorReady(ctx context.Context) error {
	if err := kc.crdReady(ctx, "msrs.msr.mirantis.com"); err != nil {
		return err
	}

	return kc.deploymentReady(ctx, constant.MSROperatorDeploymentLabels)
}

type WaitOption func(*waitOptions)

type waitOptions struct {
	numRetries int
	interval   time.Duration
}

// gatherOptions gathers custom WaitOption's and sets default values for the
// options.
func gatherOptions(opts []WaitOption) *waitOptions {
	options := &waitOptions{
		numRetries: 120,
		interval:   5 * time.Second,
	}

	for _, o := range opts {
		o(options)
	}

	return options
}

// WithCustomWait allows customization of the amount of retries and the interval
// between those retries polling will use.
func WithCustomWait(numRetries int, interval time.Duration) WaitOption {
	return func(o *waitOptions) {
		o.numRetries = numRetries
		o.interval = interval
	}
}

var errMSRCRNotReady = errors.New("MSR CR is not ready")

// WaitForMSRCRReady waits for CR object provided to be ready by polling the
// status obtained from the given object.
func (kc *KubeClient) WaitForMSRCRReady(ctx context.Context, obj *unstructured.Unstructured, rc dynamic.ResourceInterface, options ...WaitOption) error {
	opts := gatherOptions(options)

	pollCfg := pollutil.DefaultPollfConfig(log.InfoLevel, "waiting for %q CR Ready state for up to %s", obj.GetName(), opts.interval*time.Duration(opts.numRetries))
	pollCfg.Interval = opts.interval
	pollCfg.NumRetries = opts.numRetries

	err := pollutil.Pollf(pollCfg)(func() error {
		ready, e := kc.crIsReady(ctx, obj, rc)
		if e != nil {
			return fmt.Errorf("failed to process MSR CR: %w", e)
		}
		if !ready {
			return errMSRCRNotReady
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to obtain MSR CR Ready state after %s: %w", opts.interval*time.Duration(opts.numRetries), err)
	}

	return nil
}

// ApplyMSRCR applies the given MSR CR object to the cluster, reattempting
// the operation several times if it does not succeed.
func (kc *KubeClient) ApplyMSRCR(ctx context.Context, obj *unstructured.Unstructured, resourceClient dynamic.ResourceInterface) error {
	name := obj.GetName()

	existingObj, err := kc.GetMSRCR(ctx, name, resourceClient)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("MSR CR %q not found, creating", name)
		} else {
			return fmt.Errorf("failed attempting to check for MSR CR: %w", err)
		}
	}

	pollCfg := pollutil.DefaultPollfConfig(log.InfoLevel, "Applying resource YAML")
	pollCfg.Interval = 5 * time.Second
	pollCfg.NumRetries = 6

	err = pollutil.Pollf(pollCfg)(func() error {
		if existingObj == nil {
			log.Debugf("MSR resource: %q does not yet exist, creating", name)

			_, err = resourceClient.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create MSR resource: %q: %w", name, err)
			}
		} else {
			// Set the resource version to the existing object's resource version
			// if it already exists to ensure that the update succeeds.
			obj.SetResourceVersion(existingObj.GetResourceVersion())

			log.Debugf("MSR resource: %q exists, updating", name)

			_, err = resourceClient.Update(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update MSR resource: %q: %w", name, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to apply resource YAML after: %s: %w", pollCfg.Interval*time.Duration(pollCfg.NumRetries), err)
	}

	return nil
}

// PrepareNodeForMSR updates the given node name setting the MSRNodeSelector
// on the node and removing any found Kubernetes NoExecute taints added by MKE.
func (kc *KubeClient) PrepareNodeForMSR(ctx context.Context, name string) error {
	node, err := kc.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %q: %w", name, err)
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	node.Labels[constant.MSRNodeSelector] = "true"

	// Rebuild the taints list without the NoExecute taint if found.
	taints := []corev1.Taint{}
	for _, t := range node.Spec.Taints {
		if t.Key == constant.KubernetesOrchestratorTaint && t.Value == "NoExecute" {
			continue
		}
		taints = append(taints, t)
	}

	node.Spec.Taints = taints

	_, err = kc.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node %q: %w", name, err)
	}

	return nil
}

// GetMSRResourceClient returns a dynamic client for the MSR custom resource.
//
//nolint:ireturn
func (kc *KubeClient) GetMSRResourceClient() (dynamic.ResourceInterface, error) {
	client, err := dynamic.NewForConfig(kc.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return client.Resource(schema.GroupVersionResource{
		Group:    "msr.mirantis.com",
		Version:  "v1",
		Resource: "msrs",
	}).Namespace(kc.Namespace), nil
}

type SpecItemNotFoundError struct {
	NestedPath string
}

func (e *SpecItemNotFoundError) Error() string {
	return fmt.Sprintf("MSR spec.%s not found", e.NestedPath)
}

type UnknownServiceTypeError struct {
	ServiceType string
}

func (e *UnknownServiceTypeError) Error() string {
	return fmt.Sprintf("unknown MSR spec.service.serviceType: %q", e.ServiceType)
}

var (
	errNoExternalIPOrDNS = errors.New("no external IP or DNS found for MSR node")
	errNoLoadBalancerIP  = errors.New("no LoadBalancer IP found for MSR service")
)

// MSRURL constructs an MSRURL from an obtained MSR CR and other details from
// the Kubernetes cluster.
func (kc *KubeClient) MSRURL(ctx context.Context, name string, rc dynamic.ResourceInterface) (*url.URL, error) {
	msrCR, err := kc.GetMSRCR(ctx, name, rc)
	if err != nil {
		return nil, err
	}

	externalPort, found, err := unstructured.NestedInt64(msrCR.Object, "spec", "service", "externalHTTPSPort")
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR spec.service.externalHTTPSPort: %w", err)
	}

	if !found {
		return nil, &SpecItemNotFoundError{NestedPath: "service.externalHTTPSPort"}
	}

	serviceType, found, err := unstructured.NestedString(msrCR.Object, "spec", "service", "serviceType")
	if err != nil {
		return nil, fmt.Errorf("failed to get MSR spec.service.serviceType: %w", err)
	}

	if !found {
		return nil, &SpecItemNotFoundError{NestedPath: "service.serviceType"}
	}

	var (
		msrAddr string
		port    string
	)

	svc, err := kc.client.CoreV1().Services(kc.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %q: %w", name, err)
	}

	switch serviceType {
	case string(corev1.ServiceTypeNodePort):
		nodes, err := kc.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: constant.MSRNodeSelector + "=true",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list nodes: %w", err)
		}

		if len(nodes.Items) == 0 {
			return nil, &NotFoundError{Kind: "Node", Name: name}
		}

		for _, p := range svc.Spec.Ports {
			if p.Port == int32(externalPort) {
				port = strconv.Itoa(int(p.NodePort))
				break
			}
		}

		var found bool

		for _, a := range nodes.Items[0].Status.Addresses {
			// Prefer ExternalDNS over ExternalIP.
			if a.Type == corev1.NodeExternalDNS {
				msrAddr = a.Address
				found = true
				break
			} else if a.Type == corev1.NodeExternalIP {
				msrAddr = a.Address
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("%w: %q", errNoExternalIPOrDNS, nodes.Items[0].Name)
		}

	case string(corev1.ServiceTypeLoadBalancer):
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			msrAddr = svc.Status.LoadBalancer.Ingress[0].IP
		} else {
			return nil, fmt.Errorf("%w: %q", errNoLoadBalancerIP, name)
		}

	case string(corev1.ServiceTypeClusterIP):
		if svc.Spec.ClusterIP == "None" {
			// The service is headless, construct the DNS record from one
			// of the nginx pods.
			pods, err := kc.client.CoreV1().Pods(kc.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s,app.kubernetes.io/instance=%s,app.kubernetes.io/component=nginx", name, name),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list nginx pods: %w", err)
			}

			if len(pods.Items) == 0 {
				return nil, &NotFoundError{Kind: "Pod", Name: "nginx"}
			}

			msrAddr = pods.Items[0].Name + "." + kc.Namespace + ".svc.cluster.local"
		} else {
			msrAddr = svc.Spec.ClusterIP
		}

	default:
		return nil, &UnknownServiceTypeError{ServiceType: serviceType}
	}

	if port == "" {
		port = strconv.Itoa(int(externalPort))
	}

	return &url.URL{
		Scheme: "https",
		Host: func() string {
			if port == "" || port == "443" {
				return msrAddr
			}

			return msrAddr + ":" + port
		}(),
	}, nil
}
