package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Mirantis/launchpad/pkg/analytics"
	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/pkg/util/logo"
	"github.com/Mirantis/launchpad/version"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

var errInvalidArguments = errors.New("invalid arguments")

// NewApplyCommand creates new apply command to be called from cli.
func NewApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a cluster configuration",
		Flags: append(GlobalFlags, []cli.Flag{
			configFlag,
			confirmFlag,
			redactFlag,
			&cli.IntFlag{
				Name:  "concurrency",
				Usage: "Worker upgrade concurrency (number of simultaneous nodes)",
				Value: 5,
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
			&cli.BoolFlag{
				Name:  "disable-logo",
				Usage: "Disable printing of the Mirantis logo",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "force-upgrade",
				Usage: "force upgrade to run on compatible components, even if it doesn't look necessary",
				Value: false,
			},
		}...),
		Before: actions(initLogger, startUpgradeCheck, initAnalytics, checkLicense, initExec),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) (err error) {
			if ctx.Int("concurrency") < 1 {
				return fmt.Errorf("%w: invalid --concurrency %d (must be 1 or more)", errInvalidArguments, ctx.Int("concurrency"))
			}

			var logFile *os.File

			start := time.Now()
			analytics.TrackEvent("Cluster Apply Started", nil)

			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return fmt.Errorf("failed to load product config: %w", err)
			}

			defer func() {
				if err != nil && logFile != nil {
					log.Infof("See %s for more logs ", logFile.Name())
				}
			}()

			// Add logger to dump all log levels to file
			logFile, err = addFileLogger(product.ClusterName(), "apply.log")
			if err != nil {
				return fmt.Errorf("failed to add file logger: %w", err)
			}

			if isatty.IsTerminal(os.Stdout.Fd()) {
				if !ctx.Bool("disable-logo") {
					os.Stdout.WriteString(logo.Logo)
				}
				fmt.Fprintf(os.Stdout, "   Mirantis Launchpad (c) 2024 Mirantis, Inc.                          %s\n\n", version.Version)
			}

			err = product.Apply(ctx.Bool("disable-cleanup"), ctx.Bool("force"), ctx.Int("concurrency"), ctx.Bool("force-upgrade"))
			if err != nil {
				analytics.TrackEvent("Cluster Apply Failed", nil)
				return fmt.Errorf("failed to apply cluster: %w", err)
			}

			duration := time.Since(start)
			props := event.Properties{
				"duration": duration.Seconds(),
			}
			analytics.TrackEvent("Cluster Apply Completed", props)

			return nil
		},
	}
}
