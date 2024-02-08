package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	event "gopkg.in/segmentio/analytics-go.v3"
)

var reports = []string{"hosts", "mke", "msr", "config"}

func reportIsKnown(n string) bool {
	for _, v := range reports {
		if v == n {
			return true
		}
	}
	return false
}

var errInvalidReport = errors.New("invalid report")

// NewDescribeCommand creates new describe command to be called from cli.
func NewDescribeCommand() *cli.Command {
	return &cli.Command{
		Name:      "describe",
		Usage:     "Display cluster status",
		ArgsUsage: fmt.Sprintf("<%s>", strings.Join(reports, "|")),
		Flags: append(GlobalFlags, []cli.Flag{
			configFlag,
			confirmFlag,
			redactFlag,
		}...),
		Before: actions(initLogger, startUpgradeCheck, initAnalytics, checkLicense, initExec),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) error {
			report := ctx.Args().First()
			if report == "" {
				return fmt.Errorf("%w: missing report name", errInvalidReport)
			}
			if !reportIsKnown(report) {
				return fmt.Errorf("%w: unknown report %s - must be one of %s", errInvalidReport, report, strings.Join(reports, ","))
			}

			if !(ctx.Bool("debug") || ctx.Bool("trace")) {
				log.SetLevel(log.FatalLevel)
			}

			start := time.Now()
			analytics.TrackEvent("Cluster Describe Started", nil)

			product, err := config.ProductFromFile(ctx.String("config"))
			if err != nil {
				return fmt.Errorf("failed to load product config: %w", err)
			}

			err = product.Describe(ctx.Args().First())
			if err != nil {
				analytics.TrackEvent("Cluster Describe Failed", nil)
				return fmt.Errorf("failed to describe cluster: %w", err)
			}
			duration := time.Since(start)
			props := event.Properties{
				"duration": duration.Seconds(),
			}
			analytics.TrackEvent("Cluster Describe Completed", props)
			return nil
		},
	}
}
