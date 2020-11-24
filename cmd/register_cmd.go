package cmd

import (
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/cmd/register"
	"github.com/Mirantis/mcc/pkg/config/user"
	"github.com/spf13/cobra"
)

var (
	registerCmd = &cobra.Command{
		Use:   "register",
		Short: "Register a user",
		RunE: func(cmd *cobra.Command, args []string) error {

			if _, err := user.GetConfig(); err != nil {
				analytics.TrackEvent("User Not Registered", nil)
			}
			analytics.TrackEvent("User Register Started", nil)
			userConfig := &user.Config{
				Name:    name,
				Company: company,
				Email:   email,
				Eula:    acceptLicence,
			}
			err := register.Register(userConfig)
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
	name, company, email string
)

func init() {
	registerCmd.Flags().StringVarP(&name, "name", "n", "", "Your name")
	registerCmd.Flags().StringVarP(&company, "company", "c", "", "Company name")
	registerCmd.Flags().StringVarP(&email, "email", "e", "", "Email address")
	rootCmd.AddCommand(registerCmd)
}
