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

	config2 := commonapi.MCRConfig{
		Version: "25.0.10",
		Channel: "stable-25.0.10",
	}

	require.Nil(t, config2.Validate(), "unexpected sanitize error from valid MCR config")
}

func Test_ValidateNilFIPS(t *testing.T) {
	configFips := commonapi.MCRConfig{
		Version: "25.0",
		Channel: "stable-25.0/fips",
	}

	require.Nil(t, configFips.Validate(), "unexpected sanitize error from valid MCR config w/ FIPS")

	configFips2 := commonapi.MCRConfig{
		Version: "25.0.9",
		Channel: "stable-25.0.9/fips",
	}

	require.Nil(t, configFips2.Validate(), "unexpected sanitize error from valid MCR config w/ FIPS")
}

// validation should fail if version is empty (we should likely never get to such a point, but just in case)
func Test_ValidateEmptyVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrInvalidVersion, "did not receive expected error from empty MCR config version")
}

// invalid version passed, which can trigger a runtime error in GoVersion (thanks Hashicorp)
func Test_ValidateInvalidVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "this-is-not-a-valid-version",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrInvalidVersion, "did not receive expected error from invalid MCR config version")
}

// invalid version passed: reported in PRODENG-3129
func Test_ValidateInvalidVersion_PRODENG3129(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8m1-1",
		Channel: "stable-25.0.9",
	}

	require.Error(t, config.Validate(), "did not receive expected error from invalid MCR config version (PRODENG-3129)")
}

// if a full maj.min.pat version is passed, then the channel should have the full maj.min.pat part
func Test_ValidateMissingChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing the channel version")

	configFips := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable/fips", // This channel does not actually exist, but it should fail still because of the missing -25.0.8
	}

	err := configFips.Validate()
	require.ErrorIs(t, err, commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing the channel version w/ FIPS")
}

func Test_ValidateWrongChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0.9",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which has the wrong channel version")
}

func Test_ValidateWrongChannelVersionFIPS(t *testing.T) {
	configFips := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0.9/fips",
	}

	require.ErrorIs(t, configFips.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which has the wrong channel version w/ FIPS")
}

func Test_ValidateIncompleteChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0",
	}

	require.ErrorIs(t, config.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing an incomplete channel version")

	config2 := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25",
	}

	require.ErrorIs(t, config2.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing an incomplete channel version")

	config3 := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-",
	}

	require.ErrorIs(t, config3.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing an incomplete channel version")

}

func Test_ValidateIncompleteChannelVersionFIPS(t *testing.T) {
	configFips := commonapi.MCRConfig{
		Version: "25.0.8",
		Channel: "stable-25.0/fips",
	}

	require.ErrorIs(t, configFips.Validate(), commonapi.ErrChannelDoesntMatchVersion, "did not receive expected error from invalid MCR config which is missing an incomplete channel version")
}

func Test_ValidateWildcardChannelVersion(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0",
		Channel: "stable-25.0.9",
	}

	require.Nil(t, config.Validate(), "received unexpected error for valid MCR config which uses a wildcard version and specific channel")
}

func Test_ValidateWildcardChannelVersionFIPS(t *testing.T) {
	configFips := commonapi.MCRConfig{
		Version: "25.0",
		Channel: "stable-25.0.9/fips",
	}

	require.Nil(t, configFips.Validate(), "received unexpected error for valid MCR config which uses a wildcard version and specific channel w/ FIPS")
}

func Test_ValidateInternalBuild(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.12-tp1",
		Channel: "test-25.0.12",
	}

	require.Nil(t, config.Validate(), "received unexpected error for valid MCR config which contains internal build suffix -tp1")
}

func Test_ValidateInternalBuildFIPS(t *testing.T) {
	config := commonapi.MCRConfig{
		Version: "25.0.12-rc2",
		Channel: "test-25.0.12/fips",
	}

	require.Nil(t, config.Validate(), "received unexpected error for valid MCR config which contains internal build suffix -tp1 w/ FIPS")
}

