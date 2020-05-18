package util

import (
	"errors"

	"github.com/Mirantis/mcc/pkg/config"
)

// RequireRegisteredUser checks if user has registered
func RequireRegisteredUser() error {
	if _, err := config.GetUserConfig(); err != nil {
		TrackAnalyticsEvent("User not registered", nil)
		return errors.New("Registration is required. Please use `mcc register` command to register")
	}
	return nil

}
