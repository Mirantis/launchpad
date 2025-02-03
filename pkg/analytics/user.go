package analytics

import (
	"errors"

	"github.com/Mirantis/launchpad/pkg/config/user"
)

var errRegistrationRequired = errors.New("registration or license acceptance is required. please use `launchpad register` command to register")

// RequireRegisteredUser checks if user has registered.
func (c *Client) RequireRegisteredUser() error {
	if _, err := user.GetConfig(); err != nil {
		_ = c.TrackEvent("User Not Registered", nil)
		return errRegistrationRequired
	}

	return nil
}
