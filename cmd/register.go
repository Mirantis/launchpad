package cmd

import (
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/register"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/urfave/cli/v2"
)

// RegisterCommand creates register command to be called from cli
func RegisterCommand(analyticsClient *analytics.Client) *cli.Command {
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
			&cli.BoolFlag{
				Name:    "accept-license",
				Usage:   "Accept License Agreement: https://github.com/Mirantis/launchpad/blob/master/LICENSE",
				Aliases: []string{"a"},
			},
		},
		Action: func(ctx *cli.Context) error {
			if _, err := config.GetUserConfig(); err != nil {
				analyticsClient.TrackEvent("User Not Registered", nil)
			}
			analyticsClient.TrackEvent("User Register Started", nil)
			userConfig := &config.UserConfig{
				Name:    ctx.String("name"),
				Company: ctx.String("company"),
				Email:   ctx.String("email"),
				Eula:    ctx.Bool("accept-license"),
			}
			err := register.Register(userConfig, analyticsClient)
			if err == terminal.InterruptErr {
				analyticsClient.TrackEvent("User Register Cancelled", nil)
				return nil
			} else if err != nil {
				analyticsClient.TrackEvent("User Register Failed", nil)
			} else {
				analyticsClient.TrackEvent("User Register Completed", nil)
			}
			return err
		},
	}
}
