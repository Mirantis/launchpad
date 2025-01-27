package analytics

import (
	"fmt"
	"io"
	logger "log"
	"runtime"

	github.com/Mirantis/launchpad/pkg/config/user"
	github.com/Mirantis/launchpad/version"
	"github.com/denisbrodbeck/machineid"
	log "github.com/sirupsen/logrus"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

// SegmentToken is the API token we use for Segment. Set at compile time.
var SegmentToken = ""

// Analytics is the interface used for our analytics client.
type Analytics interface {
	Enqueue(msg analytics.Message) error
	Close() error
}

// Client is the struct that encapsulates the dependencies needed to send analytics
// and to interact with the analytics package.
type Client struct {
	IsEnabled       bool
	AnalyticsClient Analytics
}

var defaultClient Client

func init() {
	if SegmentToken == "" {
		defaultClient = Client{IsEnabled: false}
		return
	}
	ac, err := NewSegmentClient(SegmentToken)
	if err != nil {
		log.Warnf("failed to initialize analytics: %v", err)
		return
	}

	defaultClient.AnalyticsClient = ac
	defaultClient.IsEnabled = true
}

// NewSegmentClient returns a Segment client for uploading analytics data.
func NewSegmentClient(segmentToken string) (Analytics, error) { //nolint:ireturn
	segmentLogger := analytics.StdLogger(logger.New(io.Discard, "segment ", logger.LstdFlags))
	segmentConfig := analytics.Config{
		Logger: segmentLogger,
	}

	client, err := analytics.NewWithConfig(segmentToken, segmentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize segment client: %w", err)
	}
	return client, nil
}

// TrackEvent uploads the given event to segment if analytics tracking
// is enabled.
func (c *Client) TrackEvent(event string, properties analytics.Properties) error {
	if !c.IsEnabled {
		return nil
	}

	if properties == nil {
		properties = analytics.NewProperties()
	}

	log.Debugf("tracking analytics event '%s'", event)
	properties["os"] = runtime.GOOS
	properties["version"] = version.Version

	if err := c.AnalyticsClient.Enqueue(analytics.Track{
		UserId:      UserID(),
		AnonymousId: MachineID(),
		Event:       event,
		Properties:  properties,
	}); err != nil {
		return fmt.Errorf("failed to enqueue analytics message: %w", err)
	}
	return nil
}

// IdentifyUser identifies user on analytics service if analytics
// is enabled.
func (c *Client) IdentifyUser(userConfig *user.Config) error {
	if !c.IsEnabled {
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
	if err := c.AnalyticsClient.Enqueue(msg); err != nil {
		return fmt.Errorf("failed to enqueue analytics message: %w", err)
	}
	return nil
}

// TrackEvent uses the default analytics client to track an event.
func TrackEvent(event string, properties map[string]interface{}) {
	if err := defaultClient.TrackEvent(event, properties); err != nil {
		log.Debugf("failed to track event '%s': %v", event, err)
	}
}

// IdentifyUser uses the default analytics client to identify the user.
func IdentifyUser(userConfig *user.Config) error {
	return defaultClient.IdentifyUser(userConfig)
}

// RequireRegisteredUser uses the default analytics client to require registered user.
func RequireRegisteredUser() error {
	return defaultClient.RequireRegisteredUser()
}

// Close closes the default analytics client.
func Close() error {
	if defaultClient.AnalyticsClient != nil {
		if err := defaultClient.AnalyticsClient.Close(); err != nil {
			return fmt.Errorf("failed to close analytics client: %w", err)
		}
	}
	return nil
}

// Enabled enables the default client.
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
