package v1beta2

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1beta2 format configuration into the v1beta3 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1beta3"

	if plain["kind"] == "UCP" {
		plain["kind"] = "DockerEnterprise"
		log.Debugf("migrated v1beta2 kind: UCP to v1beta3 kind: DockerEnterprise")
	}

	// Same shape as the dispatcher's apiVersion handling: a "spec" that is
	// present but is not a mapping is malformed input, so it is reported
	// rather than silently skipped.
	if plain["spec"] != nil {
		spec, ok := plain["spec"].(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("%w: expected spec to be a mapping, got %T", migration.ErrInvalidSpec, plain["spec"])
		}
		eint, ok := spec["engine"]
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
