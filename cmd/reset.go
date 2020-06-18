package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/reset"

	"github.com/urfave/cli/v2"
)

// NewResetCommand creates new install command to be called from cli
func NewResetCommand(analyticsClient *analytics.Client) *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset (uninstall) a cluster",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to cluster config yaml",
				Aliases: []string{"c"},
				Value:   "cluster.yaml",
			},
		},
		Action: func(ctx *cli.Context) error {
			start := time.Now()
			analyticsClient.TrackEvent("Cluster Reset Started", nil)
			err := reset.Reset(ctx.String("config"), analyticsClient)
			if err != nil {
				analyticsClient.TrackEvent("Cluster Reset Failed", nil)
			} else {
				duration := time.Since(start)
				props := analytics.NewAnalyticsEventProperties()
				props["duration"] = duration.Seconds()
				analyticsClient.TrackEvent("Cluster Reset Completed", props)
			}
			return err
		},
	}
}
