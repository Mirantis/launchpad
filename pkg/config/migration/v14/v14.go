package v14

import (
	"github.com/Mirantis/mcc/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates a v1.4 format configuration into the v1.5 api format and replaces the contents of the supplied data byte slice.
func Migrate(plain map[string]interface{}) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.5"
	if spec, ok := plain["spec"].(map[interface{}]interface{}); ok {
		if mke, ok := spec["mke"].(map[interface{}]interface{}); ok {
			if swarmInstallFlags, ok := mke["swarmInstallFlags"]; ok {
				if mcr, ok := spec["mcr"].(map[interface{}]interface{}); ok {
					mcr["swarmInstallFlags"] = swarmInstallFlags
				}
				delete(mke, "swarmInstallFlags")
			}
			if SwarmUpdateCommands, ok := mke["SwarmUpdateCommands"]; ok {
				if mcr, ok := spec["mcr"].(map[interface{}]interface{}); ok {
					mcr["SwarmUpdateCommands"] = SwarmUpdateCommands
				}
				delete(mke, "SwarmUpdateCommands")
			}
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1.4 to launchpad.mirantis.com/mke/v1.5")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.4", Migrate)
}
