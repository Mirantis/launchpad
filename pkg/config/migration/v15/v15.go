package v15

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates a v1.5 format configuration into the v1.6 api format and
// replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.6"
	if spec, ok := plain["spec"].(map[interface{}]interface{}); ok {
		if msr, ok := spec["msr"].(map[interface{}]interface{}); ok {
			// Convert msr key to msr2.
			spec["msr2"] = msr
			delete(spec, "msr")
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1.5 to launchpad.mirantis.com/mke/v1.6")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.5", Migrate)
}
