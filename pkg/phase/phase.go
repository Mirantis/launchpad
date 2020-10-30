package phase

import (
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	log "github.com/sirupsen/logrus"
)

// Phase interface
type Phase interface {
	Run() error
	Title() string
	Prepare(*api.ClusterConfig) error
	ShouldRun() bool
	CleanUp()
}

// BasicPhase is a phase which has all the basic functionality like Title and default implementations for Prepare and ShouldRun
type BasicPhase struct {
	Phase

	config *api.ClusterConfig
}

// HostSelectPhase is a phase where hosts are collected before running to see if it's necessary to run the phase at all in ShouldRun
type HostSelectPhase struct {
	Phase

	config *api.ClusterConfig
	hosts  api.Hosts
}

// Prepare rceives the cluster config and stores it to the phase's config field
func (b *BasicPhase) Prepare(config *api.ClusterConfig) error {
	b.config = config
	return nil
}

// ShouldRun for BasicPhases is always true
func (b *BasicPhase) ShouldRun() bool {
	return true
}

// CleanUp basic implementation
func (b *BasicPhase) CleanUp() {}

// Title default implementation
func (h *HostSelectPhase) Title() string {
	return ""
}

// Run default implementation
func (h *HostSelectPhase) Run() error {
	return nil
}

// Prepare HostSelectPhase implementation which runs the supplied HostFilterFunc to populate the phase's hosts field
func (h *HostSelectPhase) Prepare(config *api.ClusterConfig) error {
	h.config = config
	hosts := config.Spec.Hosts.Filter(h.HostFilterFunc)
	h.hosts = hosts
	return nil
}

// ShouldRun HostSelectPhase default implementation which returns true if there are hosts that matched the HostFilterFunc
func (h *HostSelectPhase) ShouldRun() bool {
	return len(h.hosts) > 0
}

// HostFilterFunc default implementation, matches all hosts
func (h *HostSelectPhase) HostFilterFunc(host *api.Host) bool {
	return true
}

// Eventable interface
type Eventable interface {
	GetEventProperties() map[string]interface{}
}

// Analytics struct
type Analytics struct {
	EventProperties map[string]interface{}
}

// GetEventProperties returns analytic event properties
func (p *Analytics) GetEventProperties() map[string]interface{} {
	return p.EventProperties
}

// Error collects multiple error into one as we execute many phases in parallel
// for many hosts.
type Error struct {
	Errors []error
}

// AddError adds new error to the collection
func (e *Error) AddError(err error) {
	e.Errors = append(e.Errors, err)
}

// Count returns the current count of errors
func (e *Error) Count() int {
	return len(e.Errors)
}

// Error returns the combined stringified error
func (e *Error) Error() string {
	messages := []string{}
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n")
}

func runParallelOnHosts(hosts api.Hosts, config *api.ClusterConfig, action func(h *api.Host, config *api.ClusterConfig) error) error {
	return hosts.ParallelEach(func(h *api.Host) error {
		err := action(h, config)
		if err != nil {
			log.Error(err.Error())
		}
		return err
	})
}
