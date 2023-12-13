package helm

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
)

// Uninstall uninstalls a Helm release.
func (h *Helm) Uninstall(ctx context.Context, opts *Options) error {
	cfg := h.config

	u := action.NewUninstall(&cfg)

	if opts.ReleaseName == "" {
		return fmt.Errorf("release name is empty")
	}

	_, err := u.Run(opts.ReleaseName)
	return err
}
