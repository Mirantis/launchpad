package cmd

import (
	"github.com/Mirantis/mcc/pkg/create"
	"github.com/urfave/cli/v2"
)

// NewCreateCommand creates new install command to be called from cli
func NewCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Creates a new cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			err := create.Create(ctx)
			return err
		},
	}
}
