package api_test

import (
	"testing"

	commonapi "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/stretchr/testify/require"
)

func Test_ValidateNil(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0",
		Channel: "stable-25.0",
	}

	require.Nil(t, config.Validate(), "unexpected sanitize error from valid MCR config")
}

func Test_ValidateEmptyVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrInvalidVersion, "did not recieve expected error from empty MCR config version")
}

func Test_ValidateInvalidVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "this-is-not-a-valid-version",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrInvalidVersion, "did not recieve expected error from invalid MCR config version")
}

func Test_ValidateMissingChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not recieve expected error from invalid MCR config which is missing the channel version")
}

func Test_ValidateWrongChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0.9",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not recieve expected error from invalid MCR config which has the wrong channel version")
}

func Test_ValidateIncompleteChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not recieve expected error from invalid MCR config which is missing an incomplete channel version")
}

func Test_ValidateWildcardChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0",
		Channel: "stable-25.0.9",
	}

	require.Nil(t, config.Validate(), "recieved unexpected error for valid MCR config which uses a wildcard version and specific channel")
}
