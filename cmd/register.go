package cmd

import (
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/register"
	"github.com/urfave/cli/v2"
)

// RegisterCommand creates register command to be called from cli
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
		},
		Action: func(ctx *cli.Context) error {
			analytics.TrackEvent("User Register Started", nil)
			err := register.Register(ctx)
			if err == terminal.InterruptErr {
				analytics.TrackEvent("User Register Cancelled", nil)
				return nil
			} else if err != nil {
				analytics.TrackEvent("User Register Failed", nil)
			} else {
				analytics.TrackEvent("User Register Completed", nil)
			}
			return err
		},
	}
}
