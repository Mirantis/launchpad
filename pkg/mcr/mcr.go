package mcr

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

func EnsureMCRVersion(h *api.Host, specMcrVersion string) error {
	currentVersion, err := h.MCRVersion()
	if err != nil {
		if err := h.Reboot(); err != nil {
			return fmt.Errorf("%s: failed to reboot after container runtime installation: %w", h, err)
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after installation: %w", h, err)
		}
	}
	if currentVersion != specMcrVersion {
		err = h.Configurer.RestartMCR(h)
		if err != nil {
			return fmt.Errorf("%s: failed to restart container runtime: %w", h, err)
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after restart: %w", h, err)
		}
	}
	if currentVersion != specMcrVersion {
		return fmt.Errorf("%s: %w: container runtime version not %s after upgrade", h, constant.ErrVersionMismatch, specMcrVersion)
	}
	return nil
}
