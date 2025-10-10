package docker

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrNoEnvPasswordsFound = errors.New("no mathing Env U/P found")
	ErrMissingPassword     = errors.New("env username was missing matching Env password")
)

// DiscoverEnvLogin if there are any env based logins, based on a list of ENV variable prefixes.
func DiscoverEnvLogin(prefixes []string) (user, pass string, err error) {
	for _, prefix := range prefixes {
		userEnv := fmt.Sprintf("%sUSERNAME", prefix)
		passEnv := fmt.Sprintf("%sPASSWORD", prefix)

		if user = os.Getenv(userEnv); user != "" {
			pass = os.Getenv(passEnv)
			if pass == "" {
				err = fmt.Errorf("%w; %s username env variable did not have matching password variable %s", ErrMissingPassword, userEnv, passEnv)
			}
			return user, pass, err
		}
	}
	// if there were no matching vars, then we are not supposed to do a login
	err = ErrNoEnvPasswordsFound
	return user, pass, err
}
