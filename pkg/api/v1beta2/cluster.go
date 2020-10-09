package v1beta2

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Migrate migrates an v1beta2 format configuration into the v1beta3 api format and replaces the contents of the supplied data byte slice
func Migrate(data *[]byte) error {
	plain := make(map[string]interface{})
	yaml.Unmarshal(*data, &plain)

	if plain["spec"] == nil {
		return nil
	}

	eint, ok := plain["spec"].(map[interface{}]interface{})["engine"]
	if ok {
		engine := eint.(map[interface{}]interface{})
		if len(engine) > 0 {
			installURL := engine["installURL"]
			if installURL != nil {
				engine["installURLLinux"] = installURL
				delete(engine, "installURL")
				log.Debugf("migrated v1beta2 engine[installURL] to v1beta3 engine[installURLLinux]")
			}
		}
	}

	plain["apiVersion"] = "launchpad.mirantis.com/v1beta3"

	if plain["kind"] == "UCP" {
		plain["kind"] = "DockerEnterprise"
		log.Debugf("migrated v1beta2 kind: UCP to v1beta3 kind: DockerEnterprise")
	}

	log.Debugf("migrated configuration from v1beta2 to v1beta3")

	out, err := yaml.Marshal(&plain)
	if err != nil {
		return err
	}

	*data = out

	return nil
}
