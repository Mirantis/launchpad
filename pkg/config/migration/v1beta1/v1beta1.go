package v1beta1

import (
	github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1beta1 format configuration into the v1beta2 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1beta2"

	if spec, ok := plain["spec"].(map[interface{}]interface{}); ok {
		if hosts, ok := spec["hosts"].([]interface{}); ok {
			for _, h := range hosts {
				host, ok := h.(map[interface{}]interface{})
				if !ok {
					continue
				}
				ssh := make(map[string]interface{})
				host["ssh"] = ssh

				for key, val := range host {
					keyStr, ok := key.(string)
					if !ok {
						continue
					}
					switch keyStr {
					case "sshKeyPath":
						ssh["keyPath"] = val
						delete(host, key)
						log.Debugf("migrated v1beta1 host sshKeyPath '%s' to v1beta2 ssh[keyPath]", val)
					case "sshPort":
						ssh["port"] = val
						delete(host, key)
						log.Debugf("migrated v1beta1 host sshPort '%d' to v1beta2 ssh[port]", val)
					case "user":
						ssh["user"] = val
						delete(host, key)
						log.Debugf("migrated v1beta1 host user '%s' to v1beta2 ssh[user]", val)
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
