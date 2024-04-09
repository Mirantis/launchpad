package functional_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Mirantis/mcc/version"
	"github.com/stretchr/testify/assert"
)

func TestLaunchpadDownloadGreaterThanLatest(t *testing.T) {
	version.Version = "v10.5.0"
	latest := version.GetLatest(time.Second * 20)
	assert.False(t, latest.IsNewer())
}

func TestLaunchpadDownloadLessThanLatest(t *testing.T) {
	version.Version = "v1.5.0"
	latest := version.GetLatest(time.Second * 20)
	assert.True(t, latest.IsNewer())

	asset := latest.AssetForHost()
	assert.True(t, strings.Contains(asset.URL, fmt.Sprintf("https://github.com/Mirantis/launchpad/releases/download/%s", latest.TagName)))

	req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		t.Fatalf("failed to create download request %s", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create download request %s", err)
	}
	defer resp.Body.Close()

	tempFile := fmt.Sprintf("%s/launchpad", t.TempDir())
	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		t.Fatalf("failed to download: %s", err)
	}

	assert.FileExists(t, tempFile)

	t.Cleanup(func() {
		err := os.Remove(tempFile)
		if err != nil {
			t.Fatalf("failed to remove file: %s", err)
		}
	})
}

func TestLaunchpadDownloadEqualToLatest(t *testing.T) {
	latest := version.GetLatest(time.Second * 20)
	version.Version = latest.TagName
	assert.False(t, latest.IsNewer())
}
