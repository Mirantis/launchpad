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

	plain["apiVersion"] = "launchpad.mirantis.com/v1.1"

	// It gets ugly - scan for the admin username/pass in ucp/dtr installFlags and move over to ucp.username + ucp.password

	spec := plain["spec"].(map[interface{}]interface{})
	if spec["ucp"] != nil {
		ucp := spec["ucp"].(map[interface{}]interface{})
		if ucp["installFlags"] != nil {
			installFlags := ucp["installFlags"].([]interface{})
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
				log.Debugf("migrated v1 ucp.installFlags[--admin-username] to v1.1 ucp.adminUsername")
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
				log.Debugf("migrated v1 ucp.installFlags[--admin-password] to v1.1 ucp.adminPassword")
			}
		}
	}

	if spec["dtr"] != nil {
		dtr := spec["dtr"].(map[interface{}]interface{})
		if dtr["installFlags"] != nil {
			installFlags := dtr["installFlags"].([]interface{})
			drop := -1
			for idx, val := range installFlags {
				if strings.HasPrefix(val.(string), "--ucp-username") {
					user := val.(string)[strings.IndexAny(val.(string), `=" `):]
					user = strings.TrimSpace(user)
					user = strings.Trim(user, `"`)
					if user != "" {
						if spec["ucp"] == nil {
							spec["ucp"] = make(map[interface{}]interface{})
							spec["ucp"].(map[interface{}]interface{})["adminUsername"] = user
							drop = idx
						} else if spec["ucp"].(map[interface{}]interface{})["adminUsername"] == nil {
							spec["ucp"].(map[interface{}]interface{})["adminUsername"] = user
							drop = idx
						} else if spec["ucp"].(map[interface{}]interface{})["adminUsername"] != user {
							log.Warnf("spec.dtr.installFlags[--ucp-username] and spec.ucp.username mismatch")
						}
					}
				}
			}
			if drop >= 0 {
				dtr["installFlags"] = removeIndex(installFlags, drop)
				log.Debugf("migrated v1 dtr.installFlags[--ucp-username] to v1.1 ucp.adminUsername")
			}

			drop = -1
			installFlags = dtr["installFlags"].([]interface{})
			for idx, val := range installFlags {
				if strings.HasPrefix(val.(string), "--ucp-password") {
					pass := val.(string)[strings.IndexAny(val.(string), `=" `)+1:]
					pass = strings.TrimSpace(pass)
					pass = strings.Trim(pass, `"`)
					if pass != "" {
						if spec["ucp"] == nil {
							spec["ucp"] = make(map[interface{}]interface{})
							spec["ucp"].(map[interface{}]interface{})["adminPassword"] = pass
							drop = idx
						} else if spec["ucp"].(map[interface{}]interface{})["adminPassword"] == nil {
							spec["ucp"].(map[interface{}]interface{})["adminPassword"] = pass
							drop = idx
						} else if spec["ucp"].(map[interface{}]interface{})["adminPassword"] != pass {
							log.Warnf("spec.dtr.installFlags[--ucp-password] and spec.ucp.adminPassword mismatch")
						}
					}
				}
			}
			if drop >= 0 {
				dtr["installFlags"] = removeIndex(installFlags, drop)
				log.Debugf("migrated v1 dtr.installFlags[--ucp-password] to v1.1 ucp.adminPassword")
			}
		}
	}

	log.Debugf("migrated configuration from v1 to v1.1")
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
