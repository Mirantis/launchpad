package helm

import (
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
)

var errReleaseNameEmpty = errors.New("release name is empty")

// Uninstall uninstalls a Helm release.
func (h *Helm) Uninstall(opts *Options) error {
	cfg := h.config

	u := action.NewUninstall(&cfg)

	if opts.ReleaseName == "" {
		return errReleaseNameEmpty
	}

	_, err := u.Run(opts.ReleaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall Helm release %q: %w", opts.ReleaseName, err)
	}

	return nil
}
