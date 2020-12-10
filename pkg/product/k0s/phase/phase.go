package phase

import (
	"strings"

	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	log "github.com/sirupsen/logrus"
)

// BasicPhase is a phase which has all the basic functionality like Title and default implementations for Prepare and ShouldRun
type BasicPhase struct {
	Config *api.ClusterConfig
}

// Prepare rceives the cluster config and stores it to the phase's config field
func (p *BasicPhase) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	return nil
}

// ShouldRun for BasicPhases is always true
func (p *BasicPhase) ShouldRun() bool {
	return true
}

// CleanUp basic implementation
func (p *BasicPhase) CleanUp() {}

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

// RunParallelOnHosts runs a function parallelly on the listed hosts
func RunParallelOnHosts(hosts api.Hosts, config *api.ClusterConfig, action func(h *api.Host, config *api.ClusterConfig) error) error {
	return hosts.ParallelEach(func(h *api.Host) error {
		err := action(h, config)
		if err != nil {
			log.Error(err.Error())
		}
		return err
	})
}
