package cmd

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
)

// NewExecCommand creates new exec command to be called from cli
func NewExecCommand() *cli.Command {
	return &cli.Command{
		Name:      "exec",
		Usage:     "Exec a command on a host",
		ArgsUsage: "[COMMAND ..]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Path to cluster config yaml",
				Aliases:   []string{"c"},
				Value:     "launchpad.yaml",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:    "target",
				Usage:   "Target (example: address[:port])",
				Aliases: []string{"t"},
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Usage:   "Run interactive",
				Aliases: []string{"i"},
			},
			&cli.BoolFlag{
				Name:    "first",
				Usage:   "Use the first target found in configuration",
				Aliases: []string{"f"},
			},
			&cli.StringFlag{
				Name:    "role",
				Usage:   "Use the first target having this role in configuration",
				Aliases: []string{"r"},
			},
		},
		Action: func(ctx *cli.Context) error {
			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return err
			}

			args := ctx.Args().Slice()

			return product.Exec(ctx.String("address"), ctx.Bool("interactive"), ctx.Bool("first"), ctx.String("role"), shellquote.Join(args...))
		},
	}
}
