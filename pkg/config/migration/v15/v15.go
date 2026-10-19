package v15

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/config/migration"
	"github.com/Mirantis/launchpad/pkg/constant"
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
			if !ok || channels == "" {
				// Configs migrated from v1beta1/v1beta2 may have no MCR channel; use default (same as MCRConfig.SetDefaults).
				channels = constant.MCRChannel
			}

			var versions string
			switch typed := version.(type) {
			case nil: // no version: the channel is migrated as-is, including an already pinned one such as stable-25.0
			case string:
				versions = typed
			case float64: // version: 25.0  : produces a float64
				versions = fmt.Sprintf("%.1f", typed)
			case int: // version: 25  : produces an integer
				versions = fmt.Sprintf("%d.0", typed)
			default:
				log.Warnf("unclear version for migration: %+v (%T)", version, version)
			}

			if versions != "" {
				// channel might already be pinned (stable-25.0); an explicit version takes precedence over the pin
				if base, _, pinned := strings.Cut(channels, "-"); pinned {
					log.Warnf("when migrating to v1.6, both a version (%s) and a pinned channel (%s) were set; using channel %s-%s", versions, channels, base, versions)
					channels = base
				}
				channels = fmt.Sprintf("%s-%s", channels, versions)
			}
			mcr["channel"] = channels
			delete(mcr, "version")
		}
	}

	log.Debugf("migrated configuration from launchpad.mirantis.com/mke/v1.5 to launchpad.mirantis.com/mke/v1.6")
	log.Infof("Note: The configuration has been migrated from a previous version")
	log.Infof("      to see the migrated configuration use: launchpad describe config")
	return nil
}

func init() {
	migration.Register("launchpad.mirantis.com/mke/v1.5", Migrate)
}
