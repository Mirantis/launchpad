package v1beta2

import (
	github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1beta2 format configuration into the v1beta3 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1beta3"

	if plain["kind"] == "UCP" {
		plain["kind"] = "DockerEnterprise"
		log.Debugf("migrated v1beta2 kind: UCP to v1beta3 kind: DockerEnterprise")
	}

	if plain["spec"] != nil {
		eint, ok := plain["spec"].(map[interface{}]interface{})["engine"]
		if ok {
			engine, ok := eint.(map[interface{}]interface{})
			if ok && len(engine) > 0 {
				installURL := engine["installURL"]
				if installURL != nil {
					engine["installURLLinux"] = installURL
					delete(engine, "installURL")
					log.Debugf("migrated v1beta2 engine[installURL] to v1beta3 engine[installURLLinux]")
				}
			}
		}
	}

	log.Debugf("migrated configuration from v1beta2 to v1beta3")

	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/v1beta2", Migrate)
}
