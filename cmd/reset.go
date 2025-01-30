package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Mirantis/launchpad/pkg/analytics"
	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewResetCommand creates new reset command to be called from cli.
func NewResetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset (uninstall) a cluster",
		Flags: append(GlobalFlags, []cli.Flag{
			configFlag,
			confirmFlag,
			redactFlag,
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Don't ask for confirmation",
				Aliases: []string{"f"},
			},
		}...),
		Before: actions(initLogger, startUpgradeCheck, initAnalytics, checkLicense, initExec, requireForce),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) error {
			start := time.Now()
			analytics.TrackEvent("Cluster Reset Started", nil)
			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return fmt.Errorf("failed to load product config: %w", err)
			}

			err = product.Reset()
			if err != nil {
				analytics.TrackEvent("Cluster Reset Failed", nil)
				return fmt.Errorf("failed to reset cluster: %w", err)
			}

			duration := time.Since(start)
			props := event.Properties{
				"duration": duration.Seconds(),
			}
			analytics.TrackEvent("Cluster Reset Completed", props)
			return nil
		},
	}
}

var errForceRequired = errors.New("confirmation or --force required")

func requireForce(ctx *cli.Context) error {
	if !ctx.Bool("force") {
		if !isatty.IsTerminal(os.Stdout.Fd()) {
			return fmt.Errorf("%w: reset requires --force", errForceRequired)
		}
		confirmed := false
		prompt := &survey.Confirm{
			Message: "Going to reset all of the hosts, which will destroy all configuration and data, Are you sure?",
		}
		if err := survey.AskOne(prompt, &confirmed); err != nil {
			return fmt.Errorf("failed to ask for confirmation: %w", err)
		}
		if !confirmed {
			return errForceRequired
		}
	}
	return nil
}
