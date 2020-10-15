package cmd

import (
	"github.com/Mirantis/mcc/pkg/analytics"
	bundle "github.com/Mirantis/mcc/pkg/cmd/client_config"
	"github.com/urfave/cli/v2"
)

// NewClientConfigCommand creates a download bundle command to be called via the CLI
func NewClientConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "client-config",
		Usage: "Get cluster client configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Path to cluster config yaml",
				Aliases:   []string{"c"},
				Value:     "launchpad.yaml",
				TakesFile: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			err := bundle.Download(ctx.String("config"))
			if err != nil {
				analytics.TrackEvent("Client configuration download Failed", nil)
			} else {
				analytics.TrackEvent("Client configuration download Completed", nil)
			}

			return err
		},
		Before: func(ctx *cli.Context) error {
			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}
			return nil
		},
	}
}
