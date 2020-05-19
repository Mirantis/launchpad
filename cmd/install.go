package cmd

import (
	"time"

	"github.com/Mirantis/mcc/pkg/install"
	"github.com/Mirantis/mcc/pkg/util"

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
			util.TrackAnalyticsEvent("Create cluster started", nil)
			err := install.Install(ctx)
			if err != nil {
				util.TrackAnalyticsEvent("Create cluster failed", nil)
			} else {
				duration := time.Since(start)
				props := util.NewAnalyticsEventProperties()
				props["duration"] = duration.Seconds()
				util.TrackAnalyticsEvent("Create cluster succeeded", props)
			}
			return err
		},
	}
}
