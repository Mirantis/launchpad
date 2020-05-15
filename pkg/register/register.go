package register

import (
	"errors"
	"regexp"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Register ...
func Register(ctx *cli.Context) error {
	name := ctx.String("name")
	email := ctx.String("email")
	company := ctx.String("company")

	if err := validateName(name); err != nil {
		namePrompt := promptui.Prompt{
			Label:    "Name",
			Validate: validateName,
		}
		result, err := namePrompt.Run()
		if err != nil {
			return err
		}
		name = result
	}

	if err := validateEmail(email); err != nil {
		emailPrompt := promptui.Prompt{
			Label:    "Email",
			Validate: validateEmail,
		}
		result, err := emailPrompt.Run()
		if err != nil {
			return err
		}
		email = result
	}

	if company == "" {
		companyPrompt := promptui.Prompt{
			Label: "Company",
		}
		result, err := companyPrompt.Run()
		if err != nil {
			return err
		}
		company = result
	}

	userConfig := config.UserConfig{
		Name:    name,
		Company: ctx.String("company"),
		Email:   email,
	}
	err := config.SaveUserConfig(&userConfig)
	if err == nil {
		log.Info("Register completed!")
	}
	return err
}

func validateName(input string) error {
	if len(input) < 2 {
		return errors.New("Name must have more than 2 characters")
	}
	return nil
}

func validateEmail(input string) error {
	rxEmail := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if len(input) > 254 || !rxEmail.MatchString(input) {
		return errors.New("Email is not a valid email address")
	}
	return nil
}
