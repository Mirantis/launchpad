package analytics

import (
	"io"
	logger "log"
	"runtime"

	"github.com/Mirantis/mcc/pkg/config/user"
	"github.com/Mirantis/mcc/version"
	"github.com/denisbrodbeck/machineid"
	analytics "github.com/segmentio/analytics-go/v3"
	log "github.com/sirupsen/logrus"
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
	IsEnabled       bool
	AnalyticsClient Analytics
}

var defaultClient Client

func init() {
	segmentToken := DevSegmentToken
	if version.IsProduction() {
		segmentToken = ProdSegmentToken
	}
	ac, err := NewSegmentClient(segmentToken)
	if err != nil {
		log.Warnf("failed to initialize analytics: %v", err)
		return
	}

	defaultClient.AnalyticsClient = ac
	defaultClient.IsEnabled = true
}

// NewSegmentClient returns a Segment client for uploading analytics data.
func NewSegmentClient(segmentToken string) (Analytics, error) {
	segmentLogger := analytics.StdLogger(logger.New(io.Discard, "segment ", logger.LstdFlags))
	segmentConfig := analytics.Config{
		Logger: segmentLogger,
	}

	return analytics.NewWithConfig(segmentToken, segmentConfig)
}

// TrackEvent uploads the given event to segment if analytics tracking
// is enabled.
func (c *Client) TrackEvent(event string, properties analytics.Properties) error {
	if !c.IsEnabled {
		log.Debugf("analytics disabled, not tracking event '%s'", event)
		return nil
	}

	if properties == nil {
		properties = analytics.NewProperties()
	}

	log.Debugf("tracking analytics event '%s'", event)
	properties["os"] = runtime.GOOS
	properties["version"] = version.Version

	return c.AnalyticsClient.Enqueue(analytics.Track{
		UserId:      UserID(),
		AnonymousId: MachineID(),
		Event:       event,
		Properties:  properties,
	})
}

// IdentifyUser identifies user on analytics service if analytics
// is enabled
func (c *Client) IdentifyUser(userConfig *user.Config) error {
	if !c.IsEnabled {
		log.Debug("analytics disabled, not identifying user")
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
	log.Debugf("identified analytics user %+v", msg)
	return c.AnalyticsClient.Enqueue(msg)
}

// TrackEvent uses the default analytics client to track an event
func TrackEvent(event string, properties map[string]interface{}) error {
	return defaultClient.TrackEvent(event, properties)
}

// IdentifyUser uses the default analytics client to identify the user
func IdentifyUser(userConfig *user.Config) error {
	return defaultClient.IdentifyUser(userConfig)
}

// RequireRegisteredUser uses the default analytics client to require registered user
func RequireRegisteredUser() error {
	return defaultClient.RequireRegisteredUser()
}

// Close closes the default analytics client
func Close() error {
	if defaultClient.AnalyticsClient != nil {
		return defaultClient.AnalyticsClient.Close()
	}
	return nil
}

// Enabled enables the default client
func Enabled(enabled bool) {
	defaultClient.IsEnabled = enabled
}

// MachineID hashes a machine id as an anonymized identifier for our
// analytics events.
func MachineID() string {
	id, _ := machineid.ProtectedID("launchpad")
	return id
}

// UserID returs user id for our analytics events.
func UserID() string {
	userConfig, err := user.GetConfig()
	if err != nil {
		return ""
	}

	return userConfig.Email
}
