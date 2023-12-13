package phase

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// BasicPhase is a phase which has all the basic functionality like Title and default implementations for Prepare and ShouldRun.
type BasicPhase struct {
	Config *api.ClusterConfig
}

// HostSelectPhase is a phase where hosts are collected before running to see if it's necessary to run the phase at all in ShouldRun.
type HostSelectPhase struct {
	BasicPhase
	Hosts api.Hosts
}

// KubernetesPhase is a phase which requires a populated Kubernetes and Helm client.
type KubePhase struct {
	BasicPhase
	Kube *kubeclient.KubeClient
	Helm *helm.Helm
}

// CleanupDisabling can be embedded to phases that perform in-phase cleanup
// such as when using docker run --rm.
type CleanupDisabling struct {
	disableCleanup bool
}

// DisableCleanup sets the disable cleanup flag.
func (p *CleanupDisabling) DisableCleanup() {
	p.disableCleanup = true
}

// CleanupDisabled returns true when in-phase cleanup has been disabled.
func (p *CleanupDisabling) CleanupDisabled() bool {
	return p.disableCleanup
}

// Prepare rceives the cluster config and stores it to the phase's config field.
func (p *BasicPhase) Prepare(config interface{}) error {
	if cfg, ok := config.(*api.ClusterConfig); ok {
		p.Config = cfg
	}
	return nil
}

// Prepare HostSelectPhase implementation which runs the supplied HostFilterFunc to populate the phase's hosts field.
func (p *HostSelectPhase) Prepare(config interface{}) error {
	cfg, ok := config.(*api.ClusterConfig)
	if !ok {
		return nil
	}
	p.Config = cfg
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	p.Hosts = hosts
	return nil
}

// ShouldRun HostSelectPhase default implementation which returns true if there are hosts that matched the HostFilterFunc.
func (p *HostSelectPhase) ShouldRun() bool {
	return len(p.Hosts) > 0
}

// HostFilterFunc default implementation, matches all hosts.
func (p *HostSelectPhase) HostFilterFunc(_ *api.Host) bool {
	return true
}

// Prepare KubePhase implementation which populates the phase's Kube and Helm
// fields.
func (p *KubePhase) Prepare(config interface{}) (err error) {
	p.Config = config.(*api.ClusterConfig)

	// TODO: At this time we use MKE to build a Kube and Helm client, in the
	// future we should support specifying a custom Kubernetes server and
	// credentials in config to build a client so that users do not necessarily
	// need to install MKE to use MSR3.
	if !p.Config.Spec.MKE.Metadata.Installed {
		return nil
	}

	p.Kube, p.Helm, err = mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes and Helm clients from config: %w", err)
	}

	return nil
}

// Eventable interface.
type Eventable interface {
	GetEventProperties() map[string]interface{}
}

// Analytics struct.
type Analytics struct {
	EventProperties map[string]interface{}
}

// GetEventProperties returns analytic event properties.
func (p *Analytics) GetEventProperties() map[string]interface{} {
	return p.EventProperties
}

// Error collects multiple error into one as we execute many phases in parallel
// for many hosts.
type Error struct {
	Errors []error
}

// AddError adds new error to the collection.
func (e *Error) AddError(err error) {
	e.Errors = append(e.Errors, err)
}

// Count returns the current count of errors.
func (e *Error) Count() int {
	return len(e.Errors)
}

// Error returns the combined stringified error.
func (e *Error) Error() string {
	messages := []string{}
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n")
}

// RunParallelOnHosts runs a function parallelly on the listed hosts.
func RunParallelOnHosts(hosts api.Hosts, config *api.ClusterConfig, action func(h *api.Host, config *api.ClusterConfig) error) error {
	result := hosts.ParallelEach(func(h *api.Host) error {
		err := action(h, config)
		if err != nil {
			log.Error(err.Error())
		}
		return err
	})
	if result != nil {
		return fmt.Errorf("run parallel on hosts: %w", result)
	}
	return nil
}
