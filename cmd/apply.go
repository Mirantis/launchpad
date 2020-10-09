package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/apply"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewApplyCommand creates new apply command to be called from cli
func NewApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a cluster configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "launchpad.yaml",
			},
			&cli.BoolFlag{
				Name:  "prune",
				Usage: "Automatically remove nodes that are no longer a part of cluster config yaml",
				Value: false,
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
		},
		Before: func(ctx *cli.Context) error {
			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			start := time.Now()
			analytics.TrackEvent("Cluster Apply Started", nil)

			err := apply.Apply(&apply.Options{
				Config:      ctx.String("config"),
				Prune:       ctx.Bool("prune"),
				Force:       ctx.Bool("force"),
				Debug:       ctx.Bool("debug") || ctx.Bool("trace"),
				SkipCleanup: ctx.Bool("disable-cleanup"),
			})

			if err != nil {
				analytics.TrackEvent("Cluster Apply Failed", nil)
			} else {
				duration := time.Since(start)
				props := event.Properties{
					"duration": duration.Seconds(),
				}
				analytics.TrackEvent("Cluster Apply Completed", props)
			}
			return err
		},
	}
}
