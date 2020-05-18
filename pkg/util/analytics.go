package util

import (
	"runtime"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/version"
	"github.com/denisbrodbeck/machineid"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

const (
	// ProdSegmentToken is the API token we use for Segment in production.
	ProdSegmentToken = "xxx"
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

// AnalyticsClient returns a client for uploading analytics data.
func AnalyticsClient() Analytics {
	if testClient != nil {
		return testClient
	}
	segmentToken := DevSegmentToken // TODO use also ProdSegmentToken
	segmentClient := analytics.New(segmentToken)
	return segmentClient
}

// TrackAnalyticsEvent uploads the given event to segment if analytics tracking
// is enabled in the UCP config.
func TrackAnalyticsEvent(event string, properties map[string]interface{}) error {
	client := AnalyticsClient()
	defer client.Close()
	if properties == nil {
		properties = make(map[string]interface{}, 10)
	}
	properties["os"] = runtime.GOOS
	properties["version"] = version.Version
	msg := analytics.Track{
		AnonymousId: AnalyticsMachineID(),
		Event:       event,
		Properties:  properties,
	}
	if userID := AnalyticsUserID(); userID != "" {
		msg.UserId = userID
	}
	return client.Enqueue(msg)
}

// IdentifyAnalyticsUser identifies user on analytics service
func IdentifyAnalyticsUser(userConfig *config.UserConfig) error {
	client := AnalyticsClient()
	defer client.Close()
	msg := analytics.Identify{
		AnonymousId: AnalyticsMachineID(),
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

// AnalyticsMachineID hashes a machine id as an anonymized identifier for our
// analytics events.
func AnalyticsMachineID() string {
	id, _ := machineid.ProtectedID("mcc")
	return id
}

// AnalyticsUserID returs user id for our analytics events.
func AnalyticsUserID() string {
	userConfig, _ := config.GetUserConfig()
	if userConfig != nil {
		return userConfig.Email
	}
	return ""
}
