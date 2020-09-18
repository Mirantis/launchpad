package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Mirantis/mcc/cmd"
	"github.com/Mirantis/mcc/pkg/analytics"
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

	cli.AppHelpTemplate = fmt.Sprintf(`%s
GETTING STARTED:
    https://github.com/Mirantis/launchpad/blob/master/docs/getting-started.md

SUPPORT:
    https://github.com/Mirantis/launchpad/issues
`, cli.AppHelpTemplate)

	upgradeChan := make(chan *version.LaunchpadRelease)

	app := &cli.App{
		Name:  "launchpad",
		Usage: "Mirantis Launchpad",
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
				Name:    "disable-upgrade-check",
				Usage:   "Disable upgrade check",
				Aliases: []string{"u"},
				EnvVars: []string{"DISABLE_UPGRADE_CHECK"},
			},
			&cli.BoolFlag{
				Name:    "enable-upgrade-check",
				Usage:   "Enable upgrade check",
				EnvVars: []string{"DISABLE_UPGRADE_CHECK"},
				Hidden:  true,
			},
			&cli.BoolFlag{
				Name:    "accept-license",
				Usage:   "Accept License Agreement: https://github.com/Mirantis/launchpad/blob/master/LICENSE",
				Aliases: []string{"a"},
				EnvVars: []string{"ACCEPT_LICENSE"},
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Command.Name != "download-upgrade" {
				go func() {
					if (version.IsProduction() || ctx.Bool("enable-upgrade-check")) && !ctx.Bool("disable-upgrade-check") {
						upgradeChan <- version.GetUpgrade()
					} else {
						upgradeChan <- nil
					}
				}()
			}

			initLogger(ctx)
			initAnalytics(ctx)
			return nil
		},
		After: func(ctx *cli.Context) error {
			closeAnalyticsClient()
			if ctx.Command.Name != "download-upgrade" {
				latest := <-upgradeChan
				if latest != nil {
					println(fmt.Sprintf("\nA new version (%s) of `launchpad` is available. Please visit %s or run `launchpad download-upgrade --replace` to upgrade the tool.", latest.TagName, latest.URL))
				}
			}
			return nil
		},
		Commands: []*cli.Command{
			cmd.NewApplyCommand(),
			cmd.RegisterCommand(),
			cmd.NewDownloadBundleCommand(),
			cmd.NewResetCommand(),
			cmd.NewInitCommand(),
			cmd.NewDownloadUpgradeCommand(),
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
