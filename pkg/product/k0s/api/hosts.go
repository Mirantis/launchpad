package api

import (
	"fmt"
	"strings"
	"sync"
)

//Hosts are destnation hosts
type Hosts []*Host

// First returns the first host
func (hosts *Hosts) First() *Host {
	if len(*hosts) == 0 {
		return nil
	}
	return (*hosts)[0]
}

// Last returns the last host
func (hosts *Hosts) Last() *Host {
	c := len(*hosts) - 1

	if c < 0 {
		return nil
	}

	return (*hosts)[c]
}

// Find returns the first matching Host. The finder function should return true for a Host matching the criteria.
func (hosts *Hosts) Find(filter func(h *Host) bool) *Host {
	for _, h := range *hosts {
		if filter(h) {
			return (h)
		}
	}
	return nil
}

// Filter returns a filtered list of Hosts. The filter function should return true for hosts matching the criteria.
func (hosts *Hosts) Filter(filter func(h *Host) bool) Hosts {
	result := make(Hosts, 0, len(*hosts))

	for _, h := range *hosts {
		if filter(h) {
			result = append(result, h)
		}
	}

	return result
}

// ParallelEach runs a function on every Host parallelly. The function should return nil or an error.
// Any errors will be concatenated and returned.
func (hosts *Hosts) ParallelEach(filter func(h *Host) error) error {
	var wg sync.WaitGroup
	var errors []string
	type erritem struct {
		address string
		err     error
	}
	ec := make(chan erritem, 1)

	wg.Add(len(*hosts))

	for _, h := range *hosts {
		go func(h *Host) {
			ec <- erritem{h.String(), filter(h)}
		}(h)
	}

	go func() {
		for e := range ec {
			if e.err != nil {
				errors = append(errors, fmt.Sprintf("%s: %s", e.address, e.err.Error()))
			}
			wg.Done()
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("failed on %d hosts:\n - %s", len(errors), strings.Join(errors, "\n - "))
	}

	return nil
}
