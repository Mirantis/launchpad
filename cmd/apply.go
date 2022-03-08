package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewApplyCommand creates new apply command to be called from cli
func NewApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a cluster configuration",
		Flags: append(GlobalFlags, []cli.Flag{
			configFlag,
			confirmFlag,
			redactFlag,
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"c"},
				Usage:   "Worker upgrade concurrency (percentage)",
				Value:   10,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Allow continuing in some situations where prerequisite checks fail",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:   "disable-cleanup",
				Usage:  "Do not perform cleanup on failure",
				Value:  false,
				Hidden: true,
			},
		}...),
		Before: actions(initLogger, startUpgradeCheck, initAnalytics, checkLicense, initExec),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) (err error) {
			if ctx.Int("concurrency") < 1 || ctx.Int("concurrency") > 100 {
				return fmt.Errorf("invalid --concurrency %d (must be 1..100)", ctx.Int("concurrency"))
			}

			var logFile *os.File

			start := time.Now()
			analytics.TrackEvent("Cluster Apply Started", nil)

			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return
			}

			defer func() {
				if err != nil && logFile != nil {
					log.Infof("See %s for more logs ", logFile.Name())
				}

			}()

			// Add logger to dump all log levels to file
			logFile, err = addFileLogger(product.ClusterName(), "apply.log")
			if err != nil {
				return
			}

			if isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.WriteString(util.Logo)
				os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2021 Mirantis, Inc.                          v%s\n\n", version.Version))
			}

			err = product.Apply(ctx.Bool("disable-cleanup"), ctx.Bool("force"), ctx.Int("concurrency"))

			if err != nil {
				analytics.TrackEvent("Cluster Apply Failed", nil)
			} else {
				duration := time.Since(start)
				props := event.Properties{
					"duration": duration.Seconds(),
				}
				analytics.TrackEvent("Cluster Apply Completed", props)
			}

			return
		},
	}
}
