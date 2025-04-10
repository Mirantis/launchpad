package phase

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/docker"
	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

const (
	AuthEnvPrefixGeneric = "REGISTRY_"
	AuthEnvPrefixMke     = "MKE_REGISTRY_"
	AuthEnvPrefixMsr     = "MSR_REGISTRY_"
)

type loginConfig struct {
	username string
	password string
}

// AuthenticateDocker phase implementation.
type AuthenticateDocker struct {
	phase.Analytics
	phase.BasicPhase

	logins map[string]loginConfig
}

// ShouldRun is true when registry credentials are set.
func (p *AuthenticateDocker) ShouldRun() bool {
	/**
	* Possible scenarios:
	*  NOTE: an empty imageRepo is the same as a "docker.io" value in comparisons
	*  1. There is no MSR imageRepo
	*    - if MKE specific, generic env vars exist, use them to login to the MKE imageRepo
	*  2. [or] MKE and MSR both have the same imageRepo value
	*    - if MKE specific, or MSR Specific or generic env vars exist, use them to login to the MKE imageRepo
	*  3. [or] MKE and MSR both have imageRepos that are different, so two logins might be required
	*    - if MKE specific, or generic env vars exist, use them to login to the MKE imageRepo
	*    - [and] if MSR specific, or generic env vars exist, use them to login to the MSR imageRepo
	 */

	p.logins = map[string]loginConfig{} // registry keyed map

	// validate the login info for the various scenarios, collecting sets of logins required on each host.
	var discoverLoginErr error
	if p.Config.Spec.MSR == nil { //nolint:gocritic
		// check if there is an MKE specific, or a generic set of login information.
		discoverLoginErr = addLogin(p.logins, p.Config.Spec.MKE.ImageRepo, []string{AuthEnvPrefixMke, AuthEnvPrefixGeneric})
	} else if docker.CompareRepos(p.Config.Spec.MKE.ImageRepo, p.Config.Spec.MSR.ImageRepo) {
		// check if there is either an MKE, MSR or generic set of login information.
		discoverLoginErr = addLogin(p.logins, p.Config.Spec.MKE.ImageRepo, []string{AuthEnvPrefixMke, AuthEnvPrefixMsr, AuthEnvPrefixGeneric})
	} else {
		// check if there is a product or generic set of login information for each of the products.
		discoverLoginErr = errors.Join(
			addLogin(p.logins, p.Config.Spec.MKE.ImageRepo, []string{AuthEnvPrefixMke, AuthEnvPrefixGeneric}),
			addLogin(p.logins, p.Config.Spec.MSR.ImageRepo, []string{AuthEnvPrefixMsr, AuthEnvPrefixGeneric}),
		)
	}
	if discoverLoginErr != nil {
		log.Errorf("error occurred discovering registry auth values: %s", discoverLoginErr.Error())
	}

	return len(p.logins) > 0
}

// Title for the phase.
func (p *AuthenticateDocker) Title() string {
	return "Authenticate docker"
}

// Run authenticates docker on hosts.
func (p *AuthenticateDocker) Run() error {
	// now run logins to each required registry on each of the hosts.
	if err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, func(h *api.Host, _ *api.ClusterConfig) error {
		errs := []error{}
		for repo, lc := range p.logins { // running sequentially shouldn't be a problem for perfomance.
			log.Infof("%s: authenticating docker for image repo %s", h, repo)
			if err := h.Configurer.AuthenticateDocker(h, lc.username, lc.password, repo); err != nil {
				errs = append(errs, fmt.Errorf("%s: host docker authentication failed: %w", h, err))
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("docker authentication failed on at least one host: %w", err)
	}
	return nil
}

// discover if there are any env based logins for the repo, based on a list of ENV variable prefixes, and build up the list of logins needed.
func addLogin(logins map[string]loginConfig, repo string, prefixes []string) error {
	user, pass, discoverErr := docker.DiscoverEnvLogin(prefixes)
	if discoverErr != nil {
		if errors.Is(discoverErr, docker.ErrNoEnvPasswordsFound) {
			// if no envs were found for any of the prefixes, then we are not supposed to login for this repo
			return nil
		}
		return discoverErr //nolint:wrapcheck
	}

	if strings.HasPrefix(repo, "docker.io/") { // docker.io is a special case for auth
		// empty the value here so that we don't add docker.io more than once.
		repo = ""
	}

	logins[repo] = loginConfig{username: user, password: pass} // we found one, so add it to the list.

	return nil
}
