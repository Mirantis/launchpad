package v1

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1 format configuration into the v1.1 api format and replaces the contents of the supplied data byte slice
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1.1"
	log.Debugf("migrated configuration from v1 to v1.1")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/v1", Migrate)
}
