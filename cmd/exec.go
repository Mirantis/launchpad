package cmd

import (
	"github.com/Mirantis/mcc/pkg/cmd/exec"
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
				Name:    "address",
				Usage:   "Host address[:port]",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Usage:   "Run interactive",
				Aliases: []string{"i"},
			},
			&cli.BoolFlag{
				Name:    "first",
				Usage:   "Use the first host found in configuration",
				Aliases: []string{"f"},
			},
			&cli.StringFlag{
				Name:    "role",
				Usage:   "Use the first host having this role in configuration",
				Aliases: []string{"r"},
			},
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()

			return exec.Exec(ctx.String("config"), ctx.String("address"), ctx.Bool("interactive"), ctx.Bool("first"), ctx.String("role"), shellquote.Join(args...))
		},
	}
}
