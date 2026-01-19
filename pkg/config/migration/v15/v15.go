package v15

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/config/migration"
	log "github.com/sirupsen/logrus"
)

// Migrate migrates a v1.5 format configuration into the v1.6 api format and replaces the contents of the supplied data byte slice.
//
//	v1.6 uses only an MCR channel, dropping the MCR Version value.  A migrated channel can be made from the combined v1.5 channel-version
func Migrate(plain map[string]any) error {
	plain["apiVersion"] = "launchpad.mirantis.com/mke/v1.6"
	if spec, ok := plain["spec"].(map[any]any); ok {
		if mcr, ok := spec["mcr"].(map[any]any); ok {
			version := mcr["version"] // if "version: 25.0" it could be parsed as a float64 instead of a string
			channel := mcr["channel"]

			channels, ok := channel.(string)
			if !ok {
				return fmt.Errorf("could not migrate non-string channel, expected something like 'test' or 'stable-25.0`, but got '%s'", channel) // this is not likely
			}

			versions, ok := version.(string)
			if !ok { // handle some frustrating yaml parsing issues for version v`alues
				if versionf, ok := version.(float64); ok { // version: 25.0  : produces`` a float64
					versions = fmt.Sprintf("%.1f", versionf)
				} else if versioni, ok := version.(int); ok { // version: 25  : produces an integer
					versions = fmt.Sprintf("%d.0", versioni)
				} else {
					log.Warnf("unclear version for migration: %+v (%T)", version, version)
				}
			}

			// channel might already be `stable-25.0` but a version was passed, so we overrid it with a version passed
			if channelsParts := strings.Split(channels, "-"); len(channelsParts) > 1 {
				log.Warnf("when migrating to v1.6, a version (%s) and a channel (%s) were passed, but we combined them to use: %s-%s", version, channel, channels, versions)
				channels = channelsParts[0]
			}

			mcr["channel"] = fmt.Sprintf("%s-%s", channels, versions)
			delete(mcr, "version")
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/v1.4 to launchpad.mirantis.com/mke/v1.5")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.5", Migrate)
}
