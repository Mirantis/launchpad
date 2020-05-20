package cmd

import (
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/bundle"
	"github.com/urfave/cli/v2"
)

// NewDownloadBundleCommand creates a download bundle command to be called via the CLI
func NewDownloadBundleCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-bundle",
		Usage: "Download a client bundle",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "username",
				Usage:    "Username",
				Aliases:  []string{"u"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "password",
				Usage:    "Password",
				Aliases:  []string{"p"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			err := bundle.Download(ctx)
			if err != nil {
				analytics.TrackEvent("Bundle Download Failed", nil)
			} else {
				analytics.TrackEvent("Downloading bundle succeeded", nil)
			}

			return err
		},
	}
}
