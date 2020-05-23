package analytics

import (
	"os"
	"runtime"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/version"
	"github.com/denisbrodbeck/machineid"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

const (
	// ProdSegmentToken is the API token we use for Segment in production.
	ProdSegmentToken = "FlDwKhRvN6ts7GMZEgoCEghffy9HXu8Z"
	// DevSegmentToken is the API token we use for Segment in development.
	DevSegmentToken = "DLJn53HXEhUHZ4fPO45MMUhvbHRcfkLE"
)

// Analytics is the interface used for our analytics client.
type Analytics interface {
	Enqueue(msg analytics.Message) error
	Close() error
}

// testClient is only meant to be used in unit testing.
var testClient Analytics

// Client returns a client for uploading analytics data.
func Client() Analytics {
	if testClient != nil {
		return testClient
	}
	var segmentToken string
	segmentToken := DevSegmentToken
	if version.IsProduction() {
		segmentToken = ProdSegmentToken
	}
	segmentClient := analytics.New(segmentToken)
	return segmentClient
}

// IsAnalyticsDisabled detects if analytics is disabled
func IsAnalyticsDisabled() bool {
	return os.Getenv("ANALYTICS_DISABLED") == "true"
}

// TrackEvent uploads the given event to segment if analytics tracking
// is enabled.
func TrackEvent(event string, properties map[string]interface{}) error {
	if IsAnalyticsDisabled() {
		return nil
	}
	client := Client()
	defer client.Close()
	if properties == nil {
		properties = make(map[string]interface{}, 10)
	}
	properties["os"] = runtime.GOOS
	properties["version"] = version.Version
	msg := analytics.Track{
		AnonymousId: MachineID(),
		Event:       event,
		Properties:  properties,
	}
	if userID := UserID(); userID != "" {
		msg.UserId = userID
	}
	return client.Enqueue(msg)
}

// IdentifyUser identifies user on analytics service if analytics
// is enabled
func IdentifyUser(userConfig *config.UserConfig) error {
	if IsAnalyticsDisabled() {
		return nil
	}
	client := Client()
	defer client.Close()
	msg := analytics.Identify{
		AnonymousId: MachineID(),
		UserId:      userConfig.Email,
		Traits: analytics.NewTraits().
			SetName(userConfig.Name).
			SetEmail(userConfig.Email).
			Set("company", userConfig.Company),
	}
	return client.Enqueue(msg)
}

// NewAnalyticsEventProperties constructs new properties map and returns it
func NewAnalyticsEventProperties() map[string]interface{} {
	return make(map[string]interface{}, 10)
}

// MachineID hashes a machine id as an anonymized identifier for our
// analytics events.
func MachineID() string {
	id, _ := machineid.ProtectedID("launchpad")
	return id
}

// UserID returs user id for our analytics events.
func UserID() string {
	userConfig, _ := config.GetUserConfig()
	if userConfig != nil {
		return userConfig.Email
	}
	return ""
}
