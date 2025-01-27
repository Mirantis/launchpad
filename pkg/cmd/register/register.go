package register

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	github.com/Mirantis/launchpad/pkg/analytics"
	github.com/Mirantis/launchpad/pkg/config/user"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrEULADeclined is an error returned when user declines EULA.
	ErrEULADeclined = errors.New("EULA declined")
	errEULA         = errors.New("EULA check failed")
)

// Register ...
func Register(userConfig *user.Config) error {
	icons := func(icons *survey.IconSet) {
		icons.Question.Text = ">"
	}

	if validateName(userConfig.Name) != nil {
		err := survey.AskOne(&survey.Input{Message: "Name"}, &userConfig.Name, survey.WithValidator(validateName), survey.WithIcons(icons))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}

	if validateEmail(userConfig.Email) != nil {
		err := survey.AskOne(&survey.Input{Message: "Email"}, &userConfig.Email, survey.WithValidator(validateEmail), survey.WithIcons(icons))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}

	if userConfig.Company == "" {
		err := survey.AskOne(&survey.Input{Message: "Company"}, &userConfig.Company, survey.WithIcons(icons))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}

	if !userConfig.Eula {
		prompt := &survey.Confirm{
			Message: "I agree to Mirantis Launchpad Software Evaluation License Agreement https://github.com/Mirantis/launchpad/blob/master/LICENSE",
			Default: true,
		}
		err := survey.AskOne(prompt, &userConfig.Eula, survey.WithIcons(icons))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}

	if !userConfig.Eula {
		return fmt.Errorf("%w: you must agree to Mirantis Launchpad Software Evaluation License Agreement before you can use the tool", errEULA)
	}

	err := user.SaveConfig(userConfig)
	if err != nil {
		log.Error("Registration failed!")
		return fmt.Errorf("saving registration failed: %w", err)
	}
	_ = analytics.IdentifyUser(userConfig)
	log.Info("Registration completed!")
	return nil
}

func validateName(val interface{}) error {
	valStr, ok := val.(string)
	if !ok || len(valStr) < 2 {
		return fmt.Errorf("%w: name must be at least 2 characters long", errEULA)
	}
	return nil
}

func validateEmail(val interface{}) error {
	rxEmail := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	valStr, ok := val.(string)

	if !ok || len(valStr) > 254 || !rxEmail.MatchString(valStr) {
		return fmt.Errorf("%w: email is not a valid email address", errEULA)
	}
	return nil
}
