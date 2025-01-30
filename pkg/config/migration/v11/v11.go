package v11

import (
	"github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1 format configuration into the v1.1 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.2"

	if spec, ok := plain["spec"].(map[interface{}]interface{}); ok {
		if hosts, ok := spec["hosts"].([]interface{}); ok {
			for _, h := range hosts {
				host, ok := h.(map[interface{}]interface{})
				if !ok {
					continue
				}
				if ec, ok := host["engineConfig"]; ok {
					host["mcrConfig"] = ec
					delete(host, "engineConfig")
					log.Debugf("migrated v1.1 spec.hosts[*].engineConfig to v1.2 spec.hosts[*].mcrConfig")
				}
			}
		}

		if eng, ok := spec["engine"].(map[interface{}]interface{}); ok {
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
