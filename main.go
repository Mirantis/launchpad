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

var (
	analyticsClient = &analytics.Client{}
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
				Name:    "disable-analytics",
				Usage:   "Disable analytics",
				EnvVars: []string{"ANALYTICS_DISABLED"},
			},
		},
		Before: func(ctx *cli.Context) error {
			initLogger(ctx)
			initAnalytics(ctx)
			return nil
		},
		After: func(c *cli.Context) error {
			closeClient()
			version.CheckForUpgrade()
			return nil
		},
		Commands: []*cli.Command{
			cmd.NewApplyCommand(analyticsClient),
			cmd.RegisterCommand(analyticsClient),
			cmd.NewDownloadBundleCommand(analyticsClient),
			cmd.NewResetCommand(analyticsClient),
			cmd.NewInitCommand(analyticsClient),
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

const (
	// ProdSegmentToken is the API token we use for Segment in production.
	ProdSegmentToken = "FlDwKhRvN6ts7GMZEgoCEghffy9HXu8Z"
	// DevSegmentToken is the API token we use for Segment in development.
	DevSegmentToken = "DLJn53HXEhUHZ4fPO45MMUhvbHRcfkLE"
)

func initAnalytics(ctx *cli.Context) {
	segmentToken := DevSegmentToken
	if version.IsProduction() {
		segmentToken = ProdSegmentToken
	}

	var err error
	analyticsClient.AnalyticsClient, err = analytics.NewSegmentClient(segmentToken)
	if err != nil {
		analyticsClient.IsDisabled = true
	}

	analyticsClient.IsDisabled = ctx.Bool("disable-analytics")
}

func closeClient() {
	if analyticsClient.AnalyticsClient != nil {
		if err := analyticsClient.AnalyticsClient.Close(); err != nil {
			log.Debugf("Error while closing analytics client: %v", err)
		}
	}
}
