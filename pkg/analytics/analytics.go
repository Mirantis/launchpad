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

// Client is the struct that encapsulates the dependencies needed to send analytics
// and to interact with the analytics package
type Client struct {
	IsDisabled      bool
	AnalyticsClient Analytics
}

var defaultClient = Client{
	IsDisabled:      false,
	AnalyticsClient: nil,
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

// Disable disable the client
func (c *Client) Disable() {
	c.IsDisabled = true
}

// Enable enables the client
func (c *Client) Enable() {
	c.IsDisabled = false
}

// TrackEvent uses the default analytics client to track an event
func TrackEvent(event string, properties map[string]interface{}) error {
	if err := initClient(); err != nil {
		defaultClient.IsDisabled = true
	}
	return defaultClient.TrackEvent(event, properties)
}

// IdentifyUser uses the default analytics client to identify the user
func IdentifyUser(userConfig *config.UserConfig) error {
	if err := initClient(); err != nil {
		defaultClient.IsDisabled = true
	}
	return defaultClient.IdentifyUser(userConfig)
}

// RequireRegisteredUser uses the default analytics client to require registered user
func RequireRegisteredUser() error {
	if err := initClient(); err != nil {
		defaultClient.IsDisabled = true
	}
	return defaultClient.RequireRegisteredUser()
}

// Close closes the default analytics client
func Close() error {
	if defaultClient.AnalyticsClient != nil {
		return defaultClient.AnalyticsClient.Close()
	}
	return nil
}

// Disable disable the default client
func Disable() {
	defaultClient.IsDisabled = true
}

// Enable enables the default client
func Enable() {
	defaultClient.IsDisabled = false
}

func initClient() (err error) {
	if defaultClient.AnalyticsClient == nil {
		segmentToken := DevSegmentToken
		if version.IsProduction() {
			segmentToken = ProdSegmentToken
		}
		defaultClient.AnalyticsClient, err = NewSegmentClient(segmentToken)
		if err != nil {
			return err
		}
	}
	return nil
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
