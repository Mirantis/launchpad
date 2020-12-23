package v1

import (
	"regexp"
	"strings"

	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Migrate migrates an v1 format configuration into the v1.1 api format and replaces the contents of the supplied data byte slice
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.1"

	// Need to marshal back to yaml to find $VARIABLES.
	s, _ := yaml.Marshal(plain)
	re := regexp.MustCompile(`([^$])(\$[a-zA-Z_{}]{1,20})`)
	varsFound := false
	// Looping through just to give out warnings - could maybe be done in one pass using re.ReplaceAllFunc
	for _, match := range re.FindAllSubmatch(s, -1) {
		log.Warnf("found unescaped variable '%s' - migrating to api v1.1 '$%s'", match[2], match[2])
		varsFound = true
	}
	if varsFound {
		yaml.Unmarshal(re.ReplaceAll(s, []byte("$1$$$2")), plain)
	}

	var hasMsr = false

	// It gets ugly - scan for the admin username/pass in ucp/dtr installFlags and move over to ucp.username + ucp.password

	spec, ok := plain["spec"].(map[interface{}]interface{})
	if ok {
		hosts, ok := spec["hosts"]
		if ok {
			hslice := hosts.([]interface{})
			for _, h := range hslice {
				host := h.(map[interface{}]interface{})
				if host["role"].(string) == "dtr" {
					host["role"] = "msr"
					log.Debugf("changed v1 host.role[dtr] to v1.1 host.role[msr]")
					hasMsr = true
				}
			}
		}

		ucp, ok := spec["ucp"].(map[interface{}]interface{})
		if ok {
			installFlags, ok := ucp["installFlags"].([]interface{})
			if ok {
				drop := -1
				for idx, val := range installFlags {
					if strings.HasPrefix(val.(string), "--admin-username") {
						user := val.(string)[strings.IndexAny(val.(string), `=" `)+1:]
						user = strings.TrimSpace(user)
						user = strings.Trim(user, `"`)
						if user != "" {
							ucp["adminUsername"] = user
							drop = idx
						}
					}
				}
				if drop >= 0 {
					ucp["installFlags"] = removeIndex(installFlags, drop)
					log.Debugf("migrated v1 ucp.installFlags[--admin-username] to v1.1 mke.adminUsername")
				}

				drop = -1
				installFlags = ucp["installFlags"].([]interface{})
				for idx, val := range installFlags {
					if strings.HasPrefix(val.(string), "--admin-password") {
						pass := val.(string)[strings.IndexAny(val.(string), `=" `)+1:]
						pass = strings.TrimSpace(pass)
						pass = strings.Trim(pass, `"`)
						if pass != "" {
							ucp["adminPassword"] = pass
							drop = idx
						}
					}
				}
				if drop >= 0 {
					ucp["installFlags"] = removeIndex(installFlags, drop)
					log.Debugf("migrated v1 spec.ucp.installFlags[--admin-password] to v1.1 spec.mke.adminPassword")
				}
			}

			spec["mke"] = spec["ucp"]
			delete(spec, "ucp")
			log.Debugf("migrated v1 spec.ucp to v1.1 spec.mke")
		}

		dtr, ok := spec["dtr"].(map[interface{}]interface{})
		if ok {
			hasMsr = true
			installFlags, ok := dtr["installFlags"].([]interface{})
			if ok {
				drop := -1
				for idx, val := range installFlags {
					if strings.HasPrefix(val.(string), "--ucp-username") {
						user := val.(string)[strings.IndexAny(val.(string), `=" `):]
						user = strings.TrimSpace(user)
						user = strings.Trim(user, `"`)
						if user != "" {
							if spec["mke"] == nil {
								spec["mke"] = make(map[interface{}]interface{})
								spec["mke"].(map[interface{}]interface{})["adminUsername"] = user
								drop = idx
							} else if spec["mke"].(map[interface{}]interface{})["adminUsername"] == nil {
								spec["mke"].(map[interface{}]interface{})["adminUsername"] = user
								drop = idx
							} else if spec["mke"].(map[interface{}]interface{})["adminUsername"] != user {
								log.Warnf("spec.dtr.installFlags[--ucp-username] and spec.mke.adminUsername mismatch")
							}
						}
					}
				}
				if drop >= 0 {
					dtr["installFlags"] = removeIndex(installFlags, drop)
					log.Debugf("migrated v1 spec.dtr.installFlags[--ucp-username] to v1.1 spec.mke.adminUsername")
				}

				drop = -1
				installFlags = dtr["installFlags"].([]interface{})
				for idx, val := range installFlags {
					if strings.HasPrefix(val.(string), "--ucp-password") {
						pass := val.(string)[strings.IndexAny(val.(string), `=" `)+1:]
						pass = strings.TrimSpace(pass)
						pass = strings.Trim(pass, `"`)
						if pass != "" {
							if spec["mke"] == nil {
								spec["mke"] = make(map[interface{}]interface{})
								spec["mke"].(map[interface{}]interface{})["adminPassword"] = pass
								drop = idx
							} else if spec["mke"].(map[interface{}]interface{})["adminPassword"] == nil {
								spec["mke"].(map[interface{}]interface{})["adminPassword"] = pass
								drop = idx
							} else if spec["mke"].(map[interface{}]interface{})["adminPassword"] != pass {
								log.Warnf("spec.dtr.installFlags[--ucp-password] and spec.mke.adminPassword mismatch")
							}
						}
					}
				}
				if drop >= 0 {
					dtr["installFlags"] = removeIndex(installFlags, drop)
					log.Debugf("migrated v1 spec.dtr.installFlags[--ucp-password] to v1.1 spec.mke.adminPassword")
				}
			}

			if dtr["replicaConfig"] != nil {
				dtr["replicaIDs"] = dtr["replicaConfig"]
				delete(dtr, "replicaConfig")
				log.Debugf("migrated v1 spec.dtr.replicaConfig to v1.1 spec.mke.replicaIDs")
			}

			spec["msr"] = spec["dtr"]
			delete(spec, "dtr")
			log.Debugf("migrated v1 spec.dtr to v1.1 spec.msr")
		}
	}

	if plain["kind"].(string) == "DockerEnterprise" {
		if hasMsr {
			plain["kind"] = "mke+msr"
			log.Debugf("migrated v1 kind[DockerEnterprise] to v1.1 kind[mke+msr]")
		} else {
			plain["kind"] = "mke"
			log.Debugf("migrated v1 kind[DockerEnterprise] to v1.1 kind[mke]")
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1 to launchpad.mirantis.com/mke/v1.1")
	return nil
}

func removeIndex(s []interface{}, index int) []interface{} {
	ret := make([]interface{}, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func init() {
	migration.Register("launchpad.mirantis.com/v1", Migrate)
}
