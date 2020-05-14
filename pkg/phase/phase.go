package phase

import (
	"errors"
	"strings"
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// Phase interface
type Phase interface {
	Run(config *config.ClusterConfig) *PhaseError
	Title() string
}

type PhaseError struct {
	Errors []error
}

func (e *PhaseError) AddError(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *PhaseError) Count() int {
	return len(e.Errors)
}

func (e *PhaseError) Error() string {
	messages := []string{}
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n")
}

func NewPhaseError(err string) *PhaseError {
	return &PhaseError{
		Errors: []error{errors.New(err)},
	}
}

func runParallelOnHosts(hosts []*config.Host, c *config.ClusterConfig, action func(host *config.Host, config *config.ClusterConfig) error) *PhaseError {
	var wg sync.WaitGroup
	phaseError := &PhaseError{}
	for _, host := range hosts {
		wg.Add(1)
		go func(h *config.Host) {
			defer wg.Done()
			err := action(h, c)
			if err != nil {
				phaseError.AddError(err)
				log.Errorf("%s: failed -> %s", h.Address, err.Error())
			}
		}(host)
	}
	wg.Wait()

	if phaseError.Count() > 0 {
		return phaseError
	}

	return nil
}
