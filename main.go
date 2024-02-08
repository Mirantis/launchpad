package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/Mirantis/mcc/cmd"
	"github.com/Mirantis/mcc/pkg/completion"
	"github.com/Mirantis/mcc/version"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func init() {
	log.SetOutput(os.Stdout)
}

var errUnsupportedShell = errors.New("unsupported shell")

func main() {
	versionCmd := &cli.Command{
		Name: "version",
		Action: func(ctx *cli.Context) error {
			fmt.Printf("version: %s\n", version.Version)
			fmt.Printf("commit: %s\n", version.GitCommit)
			return nil
		},
	}

	completionCmd := &cli.Command{
		Name:   "completion",
		Hidden: true,
		Description: `Generates a shell auto-completion script.

   Typical locations for the generated output are:
    - Bash: /etc/bash_completion.d/launchpad
    - Zsh: /usr/local/share/zsh/site-functions/_launchpad
    - Fish: ~/.config/fish/completions/launchpad.fish`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Usage:   "Shell to generate the script for",
				Value:   "bash",
				Aliases: []string{"s"},
				EnvVars: []string{"SHELL"},
			},
		},
		Action: func(ctx *cli.Context) error {
			switch path.Base(ctx.String("shell")) {
			case "bash":
				fmt.Print(completion.BashTemplate())
			case "zsh":
				fmt.Print(completion.ZshTemplate())
			case "fish":
				t, err := ctx.App.ToFishCompletion()
				if err != nil {
					return fmt.Errorf("failed to generate fish completion: %w", err)
				}
				fmt.Print(t)
			default:
				return fmt.Errorf("%w: no completion script available for %s", errUnsupportedShell, ctx.String("shell"))
			}

			return nil
		},
	}

	cli.AppHelpTemplate = fmt.Sprintf(`%s
GETTING STARTED:
	https://docs.mirantis.com/mke/3.7/overview.html

SUPPORT:
    https://github.com/Mirantis/launchpad/issues
`, cli.AppHelpTemplate)

	app := &cli.App{
		Name:  "launchpad",
		Usage: "Mirantis Launchpad",

		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmd.NewApplyCommand(),
			cmd.RegisterCommand(),
			cmd.NewDescribeCommand(),
			cmd.NewClientConfigCommand(),
			cmd.NewExecCommand(),
			cmd.NewResetCommand(),
			cmd.NewInitCommand(),
			cmd.NewDownloadLaunchpadCommand(),
			completionCmd,
			versionCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
