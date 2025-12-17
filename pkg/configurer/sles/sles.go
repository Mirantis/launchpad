package sles

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
	"github.com/k0sproject/rig/os/registry"
)

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "sles"
		},
		func() any {
			return Configurer{}
		},
	)
}

const (
	ZYPPER_REPO_ALIAS   = "mirantis"
	ZYPPER_PACKAGE_NAME = "docker-ee"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	linux.SLES
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "socat"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	ver, verErr := configurer.ResolveLinux(h)
	if verErr != nil {
		return fmt.Errorf("could not discover Linux version information")
	}

	zypperRepoUrl := fmt.Sprintf("%s/%s/%s/%s/%s", engineConfig.RepoURL, ver.ID, "$releasever_major", "$basearch", engineConfig.Channel)
	zypperGpgUrl := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)

	// remove the repo if it exists (always recreate the repo in case our values have changes)
	if out, err := h.ExecOutput("zypper repos"); err != nil {
		return fmt.Errorf("%s: could not list zypper repos", h)
	} else if strings.Contains(out, ZYPPER_REPO_ALIAS) {
		if err := h.Exec(fmt.Sprintf("zypper removerepo %s", ZYPPER_REPO_ALIAS), exec.Sudo(h)); err != nil {
			return errors.Join(fmt.Errorf("failed to remove existing zypper MCR repo: %s", ZYPPER_REPO_ALIAS), err)
		}
	}
	log.Debugf("%s: sles MCR GPG key import %s", h, zypperGpgUrl)
	if err := h.Exec(fmt.Sprintf("sudo rpm --import %s", zypperGpgUrl), exec.Sudo(h)); err != nil {
		return errors.Join(fmt.Errorf("failed to add zypper GPG key for MCR"), err)
	}
	if err := h.Exec(fmt.Sprintf("zypper addrepo --refresh '%s' mirantis", zypperRepoUrl), exec.Sudo(h)); err != nil {
		return errors.Join(fmt.Errorf("failed to add zypper MCR repo: %s", zypperRepoUrl), err)
	}
	log.Debugf("%s: sles MCR install version", h)
	if err := c.InstallPackage(h, "docker-ee"); err != nil {
		return errors.Join(fmt.Errorf("failed to install zypper MCR packages"), err)
	}
	log.Debugf("%s: sles MCR installed from channel %s", h, engineConfig.Channel)

	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, _ string, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec("docker system prune -f"); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		if err := c.StopService(h, "docker"); err != nil {
			return fmt.Errorf("stop docker: %w", err)
		}

		if err := c.StopService(h, "containerd"); err != nil {
			return fmt.Errorf("stop containerd: %w", err)
		}

		if err := h.Exec("zypper -n remove -y --clean-deps docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
			return fmt.Errorf("remove docker-ee zypper package: %w", err)
		}
	}

	return nil
}
