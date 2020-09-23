package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mirantis/mcc/version"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

// NewDownloadUpgradeCommand creates new 'download-upgrade' command to be called from cli
func NewDownloadUpgradeCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-upgrade",
		Usage: "Download the latest launchpad version",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "replace",
				Usage:   "Replace current binary",
				Aliases: []string{"r"},
			},
		},
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

			target := "launchpad"
			if ctx.Bool("replace") {
				target, err = os.Executable()
				if err != nil {
					return err
				}
				if strings.HasSuffix(target, "exe/main") {
					return fmt.Errorf("likely running from 'go run', not replacing %s", target)
				}
			}

			f, err := os.OpenFile("launchpad", os.O_CREATE|os.O_WRONLY, 0755)
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
