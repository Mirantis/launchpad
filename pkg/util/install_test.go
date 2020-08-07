package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateImageMap(t *testing.T) {

	t.Run("Provided list of images is properly tagged as custom image repo", func(t *testing.T) {
		images := []string{
			"docker/dtr:1.2.3",
			"docker/ucp:1.2.3",
		}
		customImageRepo := "dtr.acme.com/bear"
		expectedImageMap := map[string]string{
			"docker/dtr:1.2.3": "dtr.acme.com/bear/dtr:1.2.3",
			"docker/ucp:1.2.3": "dtr.acme.com/bear/ucp:1.2.3",
		}
		actual := GenerateImageMap(images, customImageRepo)
		require.Equal(t, expectedImageMap, actual)
	})
}

func TestGetInstallFlagValue(t *testing.T) {

	t.Run("Flag value strings are properly obtained from a list of install flags", func(t *testing.T) {
		installFlags := []string{
			"--overlay-subnet 10.0.0.0/24",
			"--replica-id 123456789012",
		}
		expected := "10.0.0.0/24"
		actual := GetInstallFlagValue(installFlags, "--overlay-subnet")
		require.Equal(t, expected, actual)
	})
}
