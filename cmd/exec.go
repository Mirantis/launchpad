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
		Flags: append(GlobalFlags, []cli.Flag{
			configFlag,
			confirmFlag,
			redactFlag,
			&cli.StringSliceFlag{
				Name:    "target",
				Usage:   "Target (example: address[:port]) (can be given multiple times)",
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
			&cli.BoolFlag{
				Name:    "all",
				Usage:   "Run on all matching targets",
				Aliases: []string{"A"},
			},
			&cli.BoolFlag{
				Name:    "parallel",
				Usage:   "Run parallelly",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "role",
				Usage:   "Use targets having this role in configuration",
				Aliases: []string{"r"},
			},
			&cli.StringFlag{
				Name:    "os",
				Usage:   "Use targets running this OS ('linux', 'windows', os ID)",
				Aliases: []string{"o"},
			},
		}...),
		Before: actions(initLogger, checkLicense, initExec),
		Action: func(ctx *cli.Context) error {
			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return err
			}

			args := ctx.Args().Slice()

			return product.Exec(ctx.StringSlice("target"), ctx.Bool("interactive"), ctx.Bool("first"), ctx.Bool("all"), ctx.Bool("parallel"), ctx.String("role"), ctx.String("os"), shellquote.Join(args...))
		},
	}
}
