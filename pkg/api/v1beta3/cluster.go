package v1beta3

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Migrate migrates an v1beta3 format configuration into the v1 api format and replaces the contents of the supplied data byte slice
func Migrate(data *[]byte) error {
	plain := make(map[string]interface{})
	yaml.Unmarshal(*data, &plain)

	if plain["spec"] == nil {
		return nil
	}

	plain["apiVersion"] = "launchpad.mirantis.com/v1"
	log.Debugf("migrated configuration from v1beta3 to v1")

	out, err := yaml.Marshal(&plain)
	if err != nil {
		return err
	}

	*data = out

	return nil
}
