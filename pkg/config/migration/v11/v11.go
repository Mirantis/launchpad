package v11

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1 format configuration into the v1.1 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.2"

	spec, ok := plain["spec"].(map[interface{}]interface{})
	if ok {
		hosts, ok := spec["hosts"]
		if ok {
			hslice := hosts.([]interface{})
			for _, h := range hslice {
				host := h.(map[interface{}]interface{})
				ec, ok := host["engineConfig"]
				if ok {
					host["mcrConfig"] = ec
					delete(host, "engineConfig")
					log.Debugf("migrated v1.1 spec.hosts[*].engineConfig to v1.2 spec.hosts[*].mcrConfig")
				}
			}
		}

		eng, ok := spec["engine"].(map[interface{}]interface{})
		if ok {
			spec["mcr"] = eng
			delete(spec, "engine")
			log.Debugf("migrated v1.1 spec.engine to v1.2 spec.mcr")
		}
	}

	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.1", Migrate)
}
