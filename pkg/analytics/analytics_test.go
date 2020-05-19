package analytics

import (
	"os"
	"testing"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/segmentio/analytics-go.v3"
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
// functions sends tracking message if analytics is enabled
func TestTrackAnalyticsEvent(t *testing.T) {
	client := &mockClient{}
	testClient = client
	defer func() { testClient = nil }()
	t.Run("Analytics disabled", func(t *testing.T) {
		os.Setenv("ANALYTICS_DISABLED", "true")
		defer func() { os.Unsetenv("ANALYTICS_DISABLED") }()
		TrackEvent("test", nil)
		lastMessage := client.lastMessage
		require.Nil(t, lastMessage)
	})
	t.Run("Analytics enabled", func(t *testing.T) {
		props := make(map[string]interface{}, 1)
		props["foo"] = "bar"
		TrackEvent("test", props)
		lastMessage := client.lastMessage.(analytics.Track)
		require.NotNil(t, client.lastMessage)
		require.Equal(t, "test", lastMessage.Event)
		require.NotEmpty(t, lastMessage.AnonymousId)
		require.Equal(t, "bar", lastMessage.Properties["foo"])
	})
}

func TestIdentifyAnalyticsUser(t *testing.T) {
	client := &mockClient{}
	testClient = client
	defer func() { testClient = nil }()

	userConfig := config.UserConfig{
		Name:    "John Doe",
		Email:   "john.doe@example.org",
		Company: "Acme, Inc.",
	}
	t.Run("Analytics disabled", func(t *testing.T) {
		os.Setenv("ANALYTICS_DISABLED", "true")
		defer func() { os.Unsetenv("ANALYTICS_DISABLED") }()
		IdentifyUser(&userConfig)
		lastMessage := client.lastMessage
		require.Nil(t, lastMessage)
	})
	t.Run("Analytics enabled", func(t *testing.T) {
		IdentifyUser(&userConfig)
		lastMessage := client.lastMessage.(analytics.Identify)
		require.NotNil(t, client.lastMessage)
		require.Equal(t, "john.doe@example.org", lastMessage.UserId)
		require.Equal(t, "John Doe", lastMessage.Traits["name"])
		require.Equal(t, "john.doe@example.org", lastMessage.Traits["email"])
		require.Equal(t, "Acme, Inc.", lastMessage.Traits["company"])
	})
}
