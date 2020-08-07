package phase

import (
	"errors"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	log "github.com/sirupsen/logrus"
)

// Phase interface
type Phase interface {
	Run(config *api.ClusterConfig) error
	Title() string
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

// NewError creates new phase.Error
func NewError(err string) *Error {
	return &Error{
		Errors: []error{errors.New(err)},
	}
}

func runParallelOnHosts(hosts api.Hosts, config *api.ClusterConfig, action func(host *api.Host, config *api.ClusterConfig) error) error {
	return hosts.ParallelEach(func(host *api.Host) error {
		err := action(host, config)
		if err != nil {
			log.Error(err.Error())
		}
		return err
	})
}
