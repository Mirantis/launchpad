package k0s

import (
	"crypto/sha1"
	"fmt"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	k0s "github.com/Mirantis/mcc/pkg/product/k0s/phase"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply installs k0s on the desired host
func (p *K0s) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(
		&common.Connect{},
		&common.DetectOS{},
		&k0s.GatherFacts{},
		&common.RunHooks{Stage: "before", Action: "apply"},
		&k0s.PrepareHost{},
		&k0s.DownloadBinaries{}, // Download binaries to tempfiles for hosts that do not have a k0sBinary path but have uploadBinary: true
		&k0s.UploadBinaries{},   // Upload binaries from host's k0sBinary path (or the tempfile from above) to host
		&k0s.RunK0sDownloader{}, // For hosts that do not have k0sbinary nor uploadBinary - run the online k0s "downloader"
		&k0s.ConfigureK0s{},
		&k0s.StartK0s{},
		&common.RunHooks{Stage: "after", Action: "apply"},
		&common.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return err
	}

	clusterID := p.ClusterConfig.Spec.K0s.Metadata.ClusterID
	props := event.Properties{
		"kind":            p.ClusterConfig.Kind,
		"api_version":     p.ClusterConfig.APIVersion,
		"hosts":           len(p.ClusterConfig.Spec.Hosts),
		"k0s_version":     p.ClusterConfig.Spec.K0s.Version,
		"k0s_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}
