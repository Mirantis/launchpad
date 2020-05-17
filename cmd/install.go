package cmd

import (
	"github.com/Mirantis/mcc/pkg/install"

	"github.com/urfave/cli/v2"
)

// NewInstallCommand creates new install command to be called from cli
func NewInstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install a new cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			err := install.Install(ctx)
			return err
		},
	}
}
