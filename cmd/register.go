package cmd

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2/terminal"
	github.com/Mirantis/launchpad/pkg/analytics"
	github.com/Mirantis/launchpad/pkg/cmd/register"
	github.com/Mirantis/launchpad/pkg/config/user"
	"github.com/urfave/cli/v2"
)

// RegisterCommand creates register command to be called from cli.
func RegisterCommand() *cli.Command {
	return &cli.Command{
		Name:  "register",
		Usage: "Register a user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Usage:   "Name",
				Aliases: []string{"n"},
			},
			&cli.StringFlag{
				Name:    "company",
				Usage:   "Company",
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "email",
				Usage:   "Email",
				Aliases: []string{"e"},
			},
			debugFlag,
			traceFlag,
			telemetryFlag,
			upgradeFlag,
			licenseFlag,
		},
		Before: actions(initLogger, startUpgradeCheck, initAnalytics),
		After:  actions(closeAnalytics, upgradeCheckResult),
		Action: func(ctx *cli.Context) error {
			if _, err := user.GetConfig(); err != nil {
				analytics.TrackEvent("User Not Registered", nil)
			}
			analytics.TrackEvent("User Register Started", nil)
			userConfig := &user.Config{
				Name:    ctx.String("name"),
				Company: ctx.String("company"),
				Email:   ctx.String("email"),
				Eula:    ctx.Bool("accept-license"),
			}
			err := register.Register(userConfig)
			if err != nil {
				switch {
				case errors.Is(err, register.ErrEULADeclined):
					analytics.TrackEvent("User Register Declined", nil)
				case errors.Is(err, terminal.InterruptErr):
					analytics.TrackEvent("User Register Cancelled", nil)
				default:
					analytics.TrackEvent("User Register Failed", nil)
					return fmt.Errorf("failed to register user: %w", err)
				}
				return nil
			}

			analytics.TrackEvent("User Register Completed", nil)
			return nil
		},
	}
}
