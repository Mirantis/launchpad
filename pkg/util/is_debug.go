package util

import (
	log "github.com/sirupsen/logrus"
)

// IsDebug returns true when in debug/trace mode
func IsDebug() bool {
	return log.IsLevelEnabled(log.DebugLevel) || log.IsLevelEnabled(log.TraceLevel)
}
