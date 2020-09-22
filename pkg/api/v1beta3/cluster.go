package v1beta3

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// MigrateToCurrent migrates an v1beta3 format configuration into the current api format and replaces the contents of the supplied data byte slice
func MigrateToCurrent(data *[]byte) error {
	plain := make(map[string]interface{})
	yaml.Unmarshal(*data, &plain)

	if plain["spec"] == nil {
		return nil
	}

	hosts, ok := plain["spec"].(map[interface{}]interface{})["hosts"]
	if ok {
		hslice := hosts.([]interface{})
		for _, h := range hslice {
			host := h.(map[interface{}]interface{})
			_, hasHooks := host["hooks"]
			if hasHooks {
				return fmt.Errorf("host hooks require apiVersion >= launchpad.mirantis.com/v1")
			}

			_, hasLocal := host["localhost"]
			if hasLocal {
				return fmt.Errorf("localhost connection requires apiVersion >= launchpad.mirantis.com/v1")
			}
		}
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
