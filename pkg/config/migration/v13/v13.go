package v13

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1.2 format configuration into the v1.3 api format and replaces the contents of the supplied data byte slice
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.4"

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1.2 to launchpad.mirantis.com/mke/v1.3")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.3", Migrate)
}
