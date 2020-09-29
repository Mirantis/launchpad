package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/reset"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewResetCommand creates new reset command to be called from cli
func NewResetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset (uninstall) a cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "launchpad.yaml",
			},
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Don't ask for confirmation",
				Aliases: []string{"f"},
			},
		},
		Action: func(ctx *cli.Context) error {
			start := time.Now()
			analytics.TrackEvent("Cluster Reset Started", nil)
			err := reset.Reset(ctx.String("config"))
			if err != nil {
				analytics.TrackEvent("Cluster Reset Failed", nil)
			} else {
				duration := time.Since(start)
				props := event.Properties{
					"duration": duration.Seconds(),
				}
				analytics.TrackEvent("Cluster Reset Completed", props)
			}
			return err
		},
		Before: func(ctx *cli.Context) error {
			if !ctx.Bool("force") {
				if !isatty.IsTerminal(os.Stdout.Fd()) {
					return fmt.Errorf("reset requires --force")
				}
				confirmed := false
				prompt := &survey.Confirm{
					Message: "Going to reset all of the hosts, which will destroy all configuration and data, Are you sure?",
				}
				survey.AskOne(prompt, &confirmed)
				if !confirmed {
					return fmt.Errorf("Confirmation or --force required to proceed")
				}
			}

			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}

			return nil
		},
	}
}
