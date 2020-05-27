package register

import (
	"errors"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// Register ...
func Register(userConfig *config.UserConfig) error {
	icons := func(icons *survey.IconSet) {
		icons.Question.Text = ">"
	}

	if validateName(userConfig.Name) != nil {
		err := survey.AskOne(&survey.Input{Message: "Name"}, &userConfig.Name, survey.WithValidator(validateName), survey.WithIcons(icons))
		if err != nil {
			return err
		}
	}

	if validateEmail(userConfig.Email) != nil {
		err := survey.AskOne(&survey.Input{Message: "Email"}, &userConfig.Email, survey.WithValidator(validateEmail), survey.WithIcons(icons))
		if err != nil {
			return err
		}
	}

	if userConfig.Company == "" {
		err := survey.AskOne(&survey.Input{Message: "Company"}, &userConfig.Company, survey.WithIcons(icons))
		if err != nil {
			return err
		}
	}

	if !userConfig.Eula {
		prompt := &survey.Confirm{
			Message: "I agree to Mirantis Launchpad Software Evaluation License Agreement https://github.com/Mirantis/launchpad/blob/master/LICENSE",
			Default: true,
		}
		err := survey.AskOne(prompt, &userConfig.Eula, survey.WithIcons(icons))
		if err != nil {
			return err
		}
	}

	if !userConfig.Eula {
		return errors.New("You must agree to Mirantis Launchpad Software Evaluation License Agreement before you can use the tool")
	}

	err := config.SaveUserConfig(userConfig)
	if err == nil {
		analytics.IdentifyUser(userConfig)
		log.Info("Registration completed!")
	} else {
		log.Error("Registration failed!")
	}
	return err
}

func validateName(val interface{}) error {
	if len(val.(string)) < 2 {
		return errors.New("Name must have more than 2 characters")
	}
	return nil
}

func validateEmail(val interface{}) error {
	rxEmail := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if len(val.(string)) > 254 || !rxEmail.MatchString(val.(string)) {
		return errors.New("Email is not a valid email address")
	}
	return nil
}
