package v1beta1

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1beta1 format configuration into the v1beta2 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1beta2"

	if plain["spec"] != nil {
		hosts, ok := plain["spec"].(map[interface{}]interface{})["hosts"]
		if ok {
			hslice := hosts.([]interface{})

			for _, h := range hslice {
				host := h.(map[interface{}]interface{})
				host["ssh"] = make(map[string]interface{})
				ssh := host["ssh"].(map[string]interface{})

				for k, v := range host {
					switch k.(string) {
					case "sshKeyPath":
						ssh["keyPath"] = v
						delete(host, k)
						log.Debugf("migrated v1beta1 host sshKeyPath '%s' to v1beta2 ssh[keyPath]", v)
					case "sshPort":
						ssh["port"] = v
						delete(host, k)
						log.Debugf("migrated v1beta1 host sshPort '%d' to v1beta2 ssh[port]", v)
					case "user":
						ssh["user"] = v
						delete(host, k)
						log.Debugf("migrated v1beta1 host user '%s' to v1beta2 ssh[user]", v)
					}
				}
			}
		}
	}

	log.Debugf("migrated v1beta1 configuration to v1beta2")

	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/v1beta1", Migrate)
}
