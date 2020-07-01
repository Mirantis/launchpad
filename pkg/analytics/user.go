package analytics

import (
	"errors"

	"github.com/Mirantis/mcc/pkg/config"
)

// RequireRegisteredUser checks if user has registered
func (c *Client) RequireRegisteredUser() error {
	if _, err := config.GetUserConfig(); err != nil {
		c.TrackEvent("User Not Registered", nil)
		return errors.New("Registration or license acceptance is required. Please use `launchpad register` command to register")
	}
	return nil

}
