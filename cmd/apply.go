package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/product"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"
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
				Name:      "config",
				Usage:     "Path to cluster config yaml",
				Aliases:   []string{"c"},
				Value:     "launchpad.yaml",
				TakesFile: true,
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

			product, err := product.GetProduct(ctx)
			if err == nil {
				if isatty.IsTerminal(os.Stdout.Fd()) {
					os.Stdout.WriteString(util.Logo)
					os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
				}

				err = product.Apply()
			}
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
