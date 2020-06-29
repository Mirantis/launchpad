package v1beta1

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// MigrateToV1Beta2 migrates an v1beta1 format configuration into v1beta2 format and replaces the contents of the supplied data byte slice
func MigrateToV1Beta2(data *[]byte) error {
	plain := make(map[string]interface{})
	yaml.Unmarshal(*data, &plain)

	if plain["spec"] == nil {
		return nil
	}

	eint := plain["spec"].(map[interface{}]interface{})["engine"]
	if eint != nil {
		engine := eint.(map[interface{}]interface{})
		if len(engine) > 0 {
			installURL := engine["installURL"]
			if installURL != nil {
				engine["installURLLinux"] = installURL
				delete(engine, "installURL")
				log.Debugf("migrated v1beta1 engine[installURL] to v1beta2 engine[installURLLinux]")
			}
		}
	}

	hosts := plain["spec"].(map[interface{}]interface{})["hosts"]
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

	plain["apiVersion"] = "launchpad.mirantis.com/v1beta2"

	out, err := yaml.Marshal(&plain)
	if err != nil {
		return err
	}

	*data = out

	return nil
}
