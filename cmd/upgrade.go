package cmd

import (
	"github.com/Mirantis/mcc/pkg/upgrade"
	"github.com/urfave/cli/v2"
)

// NewUpgradeCommand creates new upgrade sub-command
func NewUpgradeCommand() *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Upgrade existing cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			err := upgrade.Upgrade(ctx)
			return err
		},
	}
}
