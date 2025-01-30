package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/launchpad/pkg/analytics"
	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

var kinds = []string{"mke", "mke+msr"}

func kindIsKnown(n string) bool {
	for _, v := range kinds {
		if v == n {
			return true
		}
	}
	return false
}

var errUnknownConfigKind = errors.New("unknown config kind")

// NewInitCommand creates new init command to be called from cli.
func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize launchpad.yaml file",
		Flags: []cli.Flag{
			telemetryFlag,
			upgradeFlag,
			licenseFlag,
			&cli.StringFlag{
				Name:    "kind",
				Usage:   "What kind of cluster definition we'll create",
				Aliases: []string{"k"},
				Value:   "mke",
			},
		},
		Before: actions(startUpgradeCheck, initAnalytics, checkLicense, initExec),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) error {
			kind := ctx.String("kind")
			if !kindIsKnown(kind) {
				return fmt.Errorf("%w: unknown kind %s - must be one of %s", errUnknownConfigKind, kind, strings.Join(kinds, ","))
			}
			analytics.TrackEvent("Cluster Init Started", nil)

			cfg, err := config.Init(kind)
			if err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			encoder := yaml.NewEncoder(os.Stdout)
			if err = encoder.Encode(cfg); err != nil {
				return fmt.Errorf("failed to encode config: %w", err)
			}
			analytics.TrackEvent("Cluster Init Completed", nil)
			return nil
		},
	}
}
