package version_test

import (
	"fmt"
	"runtime"
	"testing"

	github.com/Mirantis/launchpad/version"
)

func TestIsDevelopment(t *testing.T) {
	version.Environment = "development"
	if version.IsProduction() {
		t.Error("Expected development, got production")
	}
}

func TestIsProduction(t *testing.T) {
	version.Environment = "production"
	if !version.IsProduction() {
		t.Error("Expected production, got development")
	}
}

func TestAssetIsForHost(t *testing.T) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	asset := version.Asset{Name: fmt.Sprintf("launchpad_%s_%s_0.0.0", os, arch)}

	if !asset.IsForHost() {
		t.Error("Expected asset to be for host")
	}
}

func TestAssetIsNotForHost(t *testing.T) {
	asset := version.Asset{Name: "launchpad_linux_badarchx64_0.0.0"}

	if asset.IsForHost() {
		t.Error("Expected asset to not be for host")
	}
}

func TestLaunchpadReleaseIsNewer(t *testing.T) {
	remote := version.LaunchpadRelease{TagName: "0.0.1"}

	if !remote.IsNewer() {
		t.Error("Expected remote to be newer")
	}
}

func TestLaunchpadReleaseIsEqual(t *testing.T) {
	remote := version.LaunchpadRelease{TagName: "0.0.0"}

	if remote.IsNewer() {
		t.Error("Expected remote to not be newer")
	}
}

func TestLaunchpadReleaseIsOlder(t *testing.T) {
	version.Version = "0.0.1"
	remote := version.LaunchpadRelease{TagName: "0.0.0"}

	if remote.IsNewer() {
		t.Error("Expected remote to not be newer")
	}
}

func TestLaunchpadReleaseInvalid(t *testing.T) {
	remote := version.LaunchpadRelease{TagName: "invalid"}

	if remote.IsNewer() {
		t.Error("Expected remote to not be newer")
	}
}
