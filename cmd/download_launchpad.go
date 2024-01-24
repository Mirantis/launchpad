package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Mirantis/mcc/version"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

// NewDownloadLaunchpadCommand creates new 'download-launchpad' command to be called from cli.
func NewDownloadLaunchpadCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-launchpad",
		Usage: "Download the latest launchpad version",
		Action: func(ctx *cli.Context) error {
			latest := version.GetLatest(time.Second * 20)
			if !latest.IsNewer() {
				return fmt.Errorf("No upgrade available")
			}
			asset := latest.AssetForHost()
			if asset == nil {
				return fmt.Errorf("No download available for the current host OS + architecture")
			}

			req, err := http.NewRequest("GET", asset.URL, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			defer resp.Body.Close()

			var ext string
			if runtime.GOOS == "windows" {
				ext = ".exe"
			}

			f, err := os.OpenFile(fmt.Sprintf("launchpad%s", ext), os.O_CREATE|os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			defer f.Close()

			bar := progressbar.DefaultBytes(
				resp.ContentLength,
				"Downloading",
			)
			io.Copy(io.MultiWriter(f, bar), resp.Body)
			return nil
		},
	}
}
