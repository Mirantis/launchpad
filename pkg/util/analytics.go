package util

import (
	"github.com/denisbrodbeck/machineid"
	"gopkg.in/segmentio/analytics-go.v3"
)

const (
	// ProdSegmentToken is the API token we use for Segment in production.
	ProdSegmentToken = "xxx"
	// DevSegmentToken is the API token we use for Segment in development.
	DevSegmentToken = "yyy"
)

// Analytics is the interface used for our analytics client.
type Analytics interface {
	Enqueue(msg analytics.Message) error
	Close() error
}

// AnalyticsClient returns a client for uploading analytics data.
func AnalyticsClient() Analytics {
	segmentToken := DevSegmentToken // TODO use also ProdSegmentToken
	segmentClient := analytics.New(segmentToken)
	return segmentClient
}

// TrackAnalyticsEvent uploads the given event to segment if analytics tracking
// is enabled in the UCP config.
func TrackAnalyticsEvent(event string, properties map[string]interface{}) error {
	userID, err := AnalyticsUserID()
	if err != nil {
		return err
	}

	client := AnalyticsClient()
	defer client.Close()
	if properties == nil {
		properties = make(map[string]interface{}, 10)
	}
	properties["machineID"] = AnalyticsMachineID()
	msg := analytics.Track{
		UserId:     userID,
		Event:      event,
		Properties: properties,
	}

	return client.Enqueue(msg)
}

// AnalyticsMachineID hashes a machine id as an anonymized identifier for our
// analytics events.
func AnalyticsMachineID() string {
	id, _ := machineid.ProtectedID("mcc")
	return id
}

// AnalyticsUserID returs user id for our analytics events.
func AnalyticsUserID() (string, error) {
	return "joe@example.org", nil // TODO Read from config
}
