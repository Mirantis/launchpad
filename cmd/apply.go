package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/apply"

	"github.com/urfave/cli/v2"
)

// NewApplyCommand creates new install command to be called from cli
func NewApplyCommand(analyticsClient *analytics.Client) *cli.Command {
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
		Action: func(ctx *cli.Context) error {
			start := time.Now()
			analyticsClient.TrackEvent("Cluster Apply Started", nil)
			err := apply.Apply(ctx.String("config"), ctx.Bool("prune"), analyticsClient)
			if err != nil {
				analyticsClient.TrackEvent("Cluster Apply Failed", nil)
			} else {
				duration := time.Since(start)
				props := analytics.NewAnalyticsEventProperties()
				props["duration"] = duration.Seconds()
				analyticsClient.TrackEvent("Cluster Apply Completed", props)
			}
			return err
		},
	}
}
