package migration

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ErrInvalidAPIVersion is returned when the configuration's apiVersion is
// absent or is not a string. Both are malformed input rather than an
// unmigratable version, so they are reported here instead of being passed on
// to fail a validator tag with a less specific message.
var ErrInvalidAPIVersion = errors.New("invalid configuration apiVersion")

// ErrInvalidSpec is returned by a migrator when the configuration's spec
// section is present but is not a mapping.
var ErrInvalidSpec = errors.New("invalid configuration spec")

var migrators = make(map[string]func(map[string]interface{}) error)

// Register is used by the migrators to register their migrate function.
func Register(apiVersion string, migrator func(map[string]interface{}) error) {
	migrators[apiVersion] = migrator
}

// Migrate will run through the migrations until there is no more migrators found and returns an error if any of the migrations fail.
func Migrate(data map[string]interface{}) error {
	for {
		raw, present := data["apiVersion"]
		if !present {
			return fmt.Errorf("%w: the key is missing", ErrInvalidAPIVersion)
		}
		apiVersion, isString := raw.(string)
		if !isString {
			return fmt.Errorf("%w: expected a string, got %T", ErrInvalidAPIVersion, raw)
		}

		// No migrator for this apiVersion means the configuration is already
		// current, which is not an error.
		migrator, ok := migrators[apiVersion]
		if migrator == nil || !ok {
			return nil
		}

		if log.IsLevelEnabled(log.TraceLevel) {
			y, _ := yaml.Marshal(&data)
			log.Tracef("migration original: %s", string(y))
		}

		if err := migrator(data); err != nil {
			return err
		}

		if log.IsLevelEnabled(log.TraceLevel) {
			y, _ := yaml.Marshal(&data)
			log.Tracef("migration result: %s", string(y))
		}
	}
}
