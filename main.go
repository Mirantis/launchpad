package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/Mirantis/mcc/cmd"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/completion"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/version"
	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

func init() {
	log.SetOutput(os.Stdout)
}

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
   	- Zsh: /usr/local/share/zsh/site-functions/_launchpad`,
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
			default:
				return fmt.Errorf("no completion script available for %s", ctx.String("shell"))
			}

			return nil
		},
	}

	cli.AppHelpTemplate = fmt.Sprintf(`%s
GETTING STARTED:
    https://github.com/Mirantis/launchpad/blob/master/docs/getting-started.md

SUPPORT:
    https://github.com/Mirantis/launchpad/issues
`, cli.AppHelpTemplate)

	app := &cli.App{
		Name:                 "launchpad",
		Usage:                "Mirantis Launchpad",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging",
				Aliases: []string{"d"},
				EnvVars: []string{"DEBUG"},
			},
			&cli.BoolFlag{
				Name:    "disable-telemetry",
				Usage:   "Disable telemetry",
				Aliases: []string{"t"},
				EnvVars: []string{"DISABLE_TELEMETRY"},
			},
			&cli.BoolFlag{
				Name:    "accept-license",
				Usage:   "Accept License Agreement: https://github.com/Mirantis/launchpad/blob/master/LICENSE",
				Aliases: []string{"a"},
				EnvVars: []string{"ACCEPT_LICENSE"},
			},
		},
		Before: func(ctx *cli.Context) error {
			initLogger(ctx)
			initAnalytics(ctx)
			return nil
		},
		After: func(c *cli.Context) error {
			closeAnalyticsClient()
			version.CheckForUpgrade()
			return nil
		},
		Commands: []*cli.Command{
			cmd.NewApplyCommand(),
			cmd.RegisterCommand(),
			cmd.NewDownloadBundleCommand(),
			cmd.NewResetCommand(),
			cmd.NewInitCommand(),
			completionCmd,
			versionCmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func initLogger(ctx *cli.Context) {
	// Enable debug logging always, we'll setup hooks later to direct logs based on level
	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard) // Send all logs to nowhere by default

	// Send logs with level >= INFO to stdout

	// stdout hook on by default of course
	log.AddHook(mcclog.NewStdoutHook(ctx.Bool("debug")))
}

func initAnalytics(ctx *cli.Context) {
	if ctx.Bool("disable-telemetry") {
		analytics.Enabled(false)
	}
}

func closeAnalyticsClient() {
	if err := analytics.Close(); err != nil {
		log.Debugf("Error while closing analytics client: %v", err)
	}
}
