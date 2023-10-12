package v13

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1.2 format configuration into the v1.3 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.4"
	if spec, ok := plain["spec"].(map[interface{}]interface{}); ok {
		if mke, ok := spec["mke"].(map[interface{}]interface{}); ok {
			if mkev, ok := mke["version"].(string); ok {
				if mkev == "" {
					mke["version"] = "3.4.0"
					log.Debugf("migration defaulted MKE version to %s as an explicit version is required on the v1.4 api", mke["version"])
				}
			}
		} else {
			spec["mke"] = map[string]string{"version": "3.4.0"}
			log.Debugf("migration defaulted MKE version to %s as an explicit version is required on the v1.4 api", mke["version"])
		}

		if msr, ok := spec["msr"].(map[interface{}]interface{}); ok {
			if msrv, ok := msr["version"].(string); !ok || msrv == "" {
				msr["version"] = "2.9.0"
				log.Debugf("migration defaulted msr version to %s as an explicit version is required on the v1.4 api", msr["version"])
			}
		} else {
			kind, ok := plain["kind"].(string)
			if ok && kind != "mke+msr" {
				if hosts, ok := spec["hosts"]; ok {
					hslice := hosts.([]interface{})
					for _, h := range hslice {
						if host, ok := h.(map[interface{}]interface{}); ok {
							if role, ok := host["role"].(string); ok && role == "msr" {
								kind = "mke+msr"
								break
							}
						}
					}
				}

				if kind == "mke+msr" {
					spec["msr"] = &(map[string]string{"version": "2.9.0"})
					log.Debugf("migration defaulted msr version to %s as an explicit version is required on the v1.4 api", msr["version"])
				}
			}
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1.2 to launchpad.mirantis.com/mke/v1.3")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.3", Migrate)
}
