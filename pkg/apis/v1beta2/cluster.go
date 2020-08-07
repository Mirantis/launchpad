package v1beta2

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// MigrateToV1Beta3 migrates an v1beta1 format configuration into v1beta3 format and replaces the contents of the supplied data byte slice
func MigrateToV1Beta3(data *[]byte) error {
	plain := make(map[string]interface{})
	yaml.Unmarshal(*data, &plain)

	if plain["spec"] == nil {
		return nil
	}

	dtr := plain["spec"].(map[interface{}]interface{})["dtr"]
	if dtr != nil {
		return fmt.Errorf("dtr requires apiVersion >= launchpad.mirantis.com/v1beta3")
	}

	plain["apiVersion"] = "launchpad.mirantis.com/v1beta3"
	log.Debugf("migrated configuration from v1beta2 to v1beta3")

	out, err := yaml.Marshal(&plain)
	if err != nil {
		return err
	}

	*data = out

	return nil
}
