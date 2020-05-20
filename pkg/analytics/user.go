package analytics

import (
	"errors"

	"github.com/Mirantis/mcc/pkg/config"
)

// RequireRegisteredUser checks if user has registered
func RequireRegisteredUser() error {
	if IsAnalyticsDisabled() {
		return nil
	}
	if _, err := config.GetUserConfig(); err != nil {
		TrackEvent("User Not Registered", nil)
		return errors.New("Registration is required. Please use `mcc register` command to register")
	}
	return nil

}
