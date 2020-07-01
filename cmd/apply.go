package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/apply"

	"github.com/urfave/cli/v2"
)

// NewApplyCommand creates new install command to be called from cli
func NewApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply a cluster configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
			&cli.BoolFlag{
				Name:  "prune",
				Usage: "Automatically remove nodes that are not anymore part of cluster config yaml",
				Value: false,
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
			err := apply.Apply(ctx.String("config"), ctx.Bool("prune"))
			if err != nil {
				analytics.TrackEvent("Cluster Apply Failed", nil)
			} else {
				duration := time.Since(start)
				props := analytics.NewAnalyticsEventProperties()
				props["duration"] = duration.Seconds()
				analytics.TrackEvent("Cluster Apply Completed", props)
			}
			return err
		},
	}
}
