package analytics

import (
	"github.com/Mirantis/mcc/pkg/config"
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

// Client returns a client for uploading analytics data.
func Client() Analytics {
	if testClient != nil {
		return testClient
	}
	segmentToken := DevSegmentToken // TODO use also ProdSegmentToken
	segmentClient := analytics.New(segmentToken)
	return segmentClient
}

// TrackEvent uploads the given event to segment if analytics tracking
// is enabled in the UCP config.
func TrackEvent(event string, properties map[string]interface{}) error {
	client := Client()
	defer client.Close()
	if properties == nil {
		properties = make(map[string]interface{}, 10)
	}
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

// IdentifyUser identifies user on analytics service
func IdentifyUser(userConfig *config.UserConfig) error {
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

// MachineID hashes a machine id as an anonymized identifier for our
// analytics events.
func MachineID() string {
	id, _ := machineid.ProtectedID("mcc")
	return id
}

// UserID returs user id for our analytics events.
func UserID() string {
	return "" // TODO Read from config
}
