package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/install"

	"github.com/urfave/cli/v2"
)

// NewInstallCommand creates new install command to be called from cli
func NewInstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install a new cluster",
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
			analytics.TrackEvent("Cluster Install Started", nil)
			err := install.Install(ctx)
			if err != nil {
				analytics.TrackEvent("Cluster Install Failed", nil)
			} else {
				duration := time.Since(start)
				props := analytics.NewAnalyticsEventProperties()
				props["duration"] = duration.Seconds()
				analytics.TrackEvent("Cluster Install Succeeded", props)
			}
			return err
		},
	}
}
