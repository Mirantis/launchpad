package analytics

import (
	"io/ioutil"
	"log"
	"runtime"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/version"
	"github.com/denisbrodbeck/machineid"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

// Analytics is the interface used for our analytics client.
type Analytics interface {
	Enqueue(msg analytics.Message) error
	Close() error
}

// Client is the struct that encapsulates the dependencies needed to send analytics 
// and to interact with the analytics package
type Client struct {
	IsDisabled      bool
	AnalyticsClient Analytics
}

// NewSegmentClient returns a Segment client for uploading analytics data.
func NewSegmentClient(segmentToken string) (Analytics, error) {
	segmentLogger := analytics.StdLogger(log.New(ioutil.Discard, "segment ", log.LstdFlags))
	segmentConfig := analytics.Config{
		Logger: segmentLogger,
	}
	segmentClient, err := analytics.NewWithConfig(segmentToken, segmentConfig)
	if err != nil {
		return nil, err
	}
	return segmentClient, nil
}

// TrackEvent uploads the given event to segment if analytics tracking
// is enabled.
func (c *Client) TrackEvent(event string, properties map[string]interface{}) error {
	if c.IsDisabled {
		return nil
	}
	// TODO: close
	// client := c.DefaultAnalyticsClient()
	// defer client.Close()
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
	return c.AnalyticsClient.Enqueue(msg)
}

// IdentifyUser identifies user on analytics service if analytics
// is enabled
func (c *Client) IdentifyUser(userConfig *config.UserConfig) error {
	if c.IsDisabled {
		return nil
	}
	// client := c.DefaultAnalyticsClient()
	// defer client.Close()
	msg := analytics.Identify{
		AnonymousId: MachineID(),
		UserId:      userConfig.Email,
		Traits: analytics.NewTraits().
			SetName(userConfig.Name).
			SetEmail(userConfig.Email).
			Set("company", userConfig.Company),
	}
	return c.AnalyticsClient.Enqueue(msg)
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
