package cmd

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/urfave/cli/v2"
)

// NewInitCommand creates new init command to be called from cli
func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize launchpad.yaml file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "kind",
				Usage:   "What kind of cluster definition we'll create",
				Aliases: []string{"k"},
				Value:   "DockerEnterprise",
			},
		},
		Action: func(ctx *cli.Context) error {
			analytics.TrackEvent("Cluster Init Started", nil)
			if err := analytics.RequireRegisteredUser(); err != nil {
				analytics.TrackEvent("Cluster Init Failed", nil)
				return err
			}

			cfg, err := config.Init(ctx.String("kind"))
			if err != nil {
				return err
			}

			encoder := yaml.NewEncoder(os.Stdout)
			err = encoder.Encode(cfg)

			if err != nil {
				analytics.TrackEvent("Cluster Init Failed", nil)
			} else {
				analytics.TrackEvent("Cluster Init Completed", nil)
			}
			return err
		},
	}
}
