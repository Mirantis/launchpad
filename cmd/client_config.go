package cmd

import (
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/product"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// NewClientConfigCommand creates a download bundle command to be called via the CLI
func NewClientConfigCommand() *cli.Command {
	return &cli.Command{
		Name:    "client-config",
		Aliases: []string{"download-bundle"},
		Usage:   "Get cluster client configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Path to cluster config yaml",
				Aliases:   []string{"c"},
				Value:     "launchpad.yaml",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:    "username",
				Usage:   "Username",
				Aliases: []string{"u"},
				Hidden:  true,
			},
			&cli.StringFlag{
				Name:    "Password",
				Usage:   "Password",
				Aliases: []string{"p"},
				Hidden:  true,
			},
		},
		Action: func(ctx *cli.Context) error {
			product, err := product.GetProduct(ctx)
			if err == nil {
				err = product.ClientConfig()
			}
			if err != nil {
				analytics.TrackEvent("Client configuration download Failed", nil)
			} else {
				analytics.TrackEvent("Client configuration download Completed", nil)
			}

			return err
		},
		Before: func(ctx *cli.Context) error {
			if strings.Contains(strings.Join(os.Args, " "), "download-bundle") {
				log.Warn()
				log.Warn("[DEPRECATED] The 'download-bundle' subcommand is now 'client-config'.")
				log.Warn()
			}
			if ctx.String("username") != "" || ctx.String("password") != "" {
				log.Warn("[DEPRECATED] The --username and --password flags are ignored and are now read from the configuration file")
				log.Warn()
			}

			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}
			return nil
		},
	}
}
