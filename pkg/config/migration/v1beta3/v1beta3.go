package v1beta3

import (
	"github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates an v1beta3 format configuration into the v1 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/v1"

	log.Debugf("migrated configuration from v1beta3 to v1")

	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/v1beta3", Migrate)
}
