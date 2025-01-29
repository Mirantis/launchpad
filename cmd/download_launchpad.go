package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Mirantis/launchpad/version"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var errNoDownloadAvailable = fmt.Errorf("no download available")

// NewDownloadLaunchpadCommand creates new 'download-launchpad' command to be called from cli.
func NewDownloadLaunchpadCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-launchpad",
		Usage: "Download the latest launchpad version",
		Action: func(_ *cli.Context) error {
			latest := version.GetLatest(time.Second * 20)
			if !latest.IsNewer() {
				return fmt.Errorf("%w: upgrade not available", errNoDownloadAvailable)
			}
			asset := latest.AssetForHost()
			if asset == nil {
				return fmt.Errorf("%w: no download available for the current host OS + architecture", errNoDownloadAvailable)
			}

			req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
			if err != nil {
				return fmt.Errorf("failed to create download request: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to perform download request: %w", err)
			}
			defer resp.Body.Close()

			var ext string
			if runtime.GOOS == "windows" {
				ext = ".exe"
			}

			f, err := os.OpenFile(fmt.Sprintf("launchpad%s", ext), os.O_CREATE|os.O_WRONLY, 0o755)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer f.Close()

			bar := progressbar.DefaultBytes(
				resp.ContentLength,
				"Downloading",
			)
			_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
			if err != nil {
				return fmt.Errorf("failed to download: %w", err)
			}
			return nil
		},
	}
}
