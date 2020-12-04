package dummy

import (
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/phase"
	dum "github.com/Mirantis/mcc/pkg/product/dummy/phase"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply - does very little
func (p *Dummy) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup

	phaseManager.AddPhases(
		&dum.Connect{},
		&dum.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return err
	}

	props := event.Properties{
		"kind":        p.ClusterConfig.Kind,
		"api_version": p.ClusterConfig.APIVersion,
		"hosts":       len(p.ClusterConfig.Spec.Hosts),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}
