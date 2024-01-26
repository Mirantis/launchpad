package migration

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var migrators = make(map[string]func(map[string]interface{}) error)

// Register is used by the migrators to register their migrate function.
func Register(apiVersion string, migrator func(map[string]interface{}) error) {
	migrators[apiVersion] = migrator
}

// Migrate will run through the migrations until there is no more migrators found and returns an error if any of the migrations fail.
func Migrate(data map[string]interface{}) error {
	for {
		migrator, ok := migrators[data["apiVersion"].(string)]
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
