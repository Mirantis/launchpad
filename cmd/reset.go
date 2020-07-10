package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/reset"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// NewResetCommand creates new install command to be called from cli
func NewResetCommand() *cli.Command {
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
			if !ctx.Bool("accept-license") {
				return analytics.RequireRegisteredUser()
			}
			return nil
		},
	}
}
