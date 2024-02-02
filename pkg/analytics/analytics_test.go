package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/Mirantis/mcc/pkg/config/user"
)

type mockClient struct {
	lastMessage analytics.Message
	closed      bool
}

func (m *mockClient) Enqueue(msg analytics.Message) error {
	m.lastMessage = msg
	return nil
}

func (m *mockClient) Close() error {
	m.closed = true
	return nil
}

// TestTrackAnalyticsEvent tests that the TrackAnalyticsEvent
// functions sends tracking message if analytics is enabled.
func TestTrackAnalyticsEvent(t *testing.T) {
	client := &mockClient{}
	analyticsClient := Client{
		IsEnabled:       true,
		AnalyticsClient: client,
	}

	t.Run("Analytics disabled", func(t *testing.T) {
		analyticsClient.IsEnabled = false
		t.Cleanup(func() { analyticsClient.IsEnabled = true })

		analyticsClient.TrackEvent("test", nil)
		lastMessage := client.lastMessage
		require.Nil(t, lastMessage)
	})
	t.Run("Analytics enabled", func(t *testing.T) {
		props := analytics.Properties{
			"foo": "bar",
		}
		analyticsClient.TrackEvent("test", props)

		var lastMessage analytics.Track

		if assert.NotNil(t, client.lastMessage) {
			lastMessage = client.lastMessage.(analytics.Track)
			require.Equal(t, "test", lastMessage.Event)
			require.NotEmpty(t, lastMessage.AnonymousId)
			require.Equal(t, "bar", lastMessage.Properties["foo"])
		}
	})
}

func TestIdentifyAnalyticsUser(t *testing.T) {
	client := &mockClient{}
	analyticsClient := Client{
		IsEnabled:       true,
		AnalyticsClient: client,
	}

	userConfig := user.Config{
		Name:    "John Doe",
		Email:   "john.doe@example.org",
		Company: "Acme, Inc.",
	}
	t.Run("Analytics disabled", func(t *testing.T) {
		analyticsClient.IsEnabled = false
		t.Cleanup(func() { analyticsClient.IsEnabled = true })

		analyticsClient.IdentifyUser(&userConfig)
		lastMessage := client.lastMessage
		require.Nil(t, lastMessage)
	})
	t.Run("Analytics enabled", func(t *testing.T) {
		analyticsClient.IdentifyUser(&userConfig)
		lastMessage := client.lastMessage.(analytics.Identify)
		require.NotNil(t, client.lastMessage)
		require.Equal(t, "john.doe@example.org", lastMessage.UserId)
		require.Equal(t, "John Doe", lastMessage.Traits["name"])
		require.Equal(t, "john.doe@example.org", lastMessage.Traits["email"])
		require.Equal(t, "Acme, Inc.", lastMessage.Traits["company"])
	})
}

func TestDefaultClient(t *testing.T) {
	userConfig := user.Config{
		Name:    "John Doe",
		Email:   "john.doe@example.org",
		Company: "Acme, Inc.",
	}

	t.Run("Analytics disabled", func(t *testing.T) {
		old := defaultClient.IsEnabled
		Enabled(false)
		defer Enabled(old)

		TrackEvent("foobar", nil)
		IdentifyUser(&userConfig)
		RequireRegisteredUser()
	})

	t.Run("Analytics enabled", func(t *testing.T) {
		old := defaultClient.IsEnabled
		Enabled(true)
		defer Enabled(old)

		TrackEvent("foobar", nil)
		IdentifyUser(&userConfig)
		RequireRegisteredUser()
	})
}
