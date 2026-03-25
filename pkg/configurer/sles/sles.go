// Package sles implements the mke/config/HostConfigurer for SLES machines
package sles

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
	"github.com/k0sproject/rig/os/registry"
	log "github.com/sirupsen/logrus"
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
	ZypperPackageNotFound = "No matching items found"
	ZypperRepoAlias       = "mirantis"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	linux.SLES
	configurer.LinuxConfigurer
}

// PrepareHost prepares the machine host by installing the needed base packages, and fixing any container issues.
func (c Configurer) PrepareHost(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "socat"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}

	if c.IsContainer(h) {
		if err := c.FixContainer(h); err != nil {
			return fmt.Errorf("fix container: %w", err)
		}
	}

	log.Debugf("%s: checking for Docker-CE conflict", h)
	if out, err := h.ExecOutput("zypper search --type=package --installed-only docker"); err == nil && !strings.Contains(out, ZypperPackageNotFound) && !strings.Contains(out, "docker-ee") {
		log.Warnf("%s: detected Docker-CE, removing from system", h)
		if err := h.Exec("zypper remove -y --clean-deps docker", exec.Sudo(h)); err != nil {
			return fmt.Errorf("could not remove existing docker-ce installation: %w", err)
		}
	}

	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, engineConfig commonconfig.MCRConfig) error {
	ver, verErr := configurer.ResolveLinux(h)
	if verErr != nil {
		return fmt.Errorf("could not discover Linux version information")
	}

	zypperRepoURL := fmt.Sprintf("%s/%s/%s/%s/%s", engineConfig.RepoURL, ver.ID, "$releasever_major", "$basearch", engineConfig.Channel)
	zypperGpgURL := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)

	// remove the repo if it exists (always recreate the repo in case our values have changes)
	if out, err := h.ExecOutput("zypper repos"); err != nil {
		return fmt.Errorf("%s: could not list zypper repos", h)
	} else if strings.Contains(out, ZypperRepoAlias) {
		if err := h.Exec(fmt.Sprintf("zypper removerepo %s", ZypperRepoAlias), exec.Sudo(h)); err != nil {
			return errors.Join(fmt.Errorf("failed to remove existing zypper MCR repo: %s", ZypperRepoAlias), err)
		}
	}
	log.Debugf("%s: sles MCR GPG key import %s", h, zypperGpgURL)
	if err := h.Exec(fmt.Sprintf("sudo rpm --import %s", zypperGpgURL), exec.Sudo(h)); err != nil {
		return errors.Join(fmt.Errorf("failed to add zypper GPG key for MCR"), err)
	}
	if err := h.Exec(fmt.Sprintf("zypper addrepo --refresh '%s' mirantis", zypperRepoURL), exec.Sudo(h)); err != nil {
		return errors.Join(fmt.Errorf("failed to add zypper MCR repo: %s", zypperRepoURL), err)
	}

	log.Debugf("%s: sles MCR install version", h)
	if err := c.InstallPackage(h, "containerd.io"); err != nil {
		return fmt.Errorf("package manager could not install containerd.io")
	}
	if err := c.InstallPackage(h, "docker-ee"); err != nil {
		return fmt.Errorf("package manager could not install docker-ee")
	}

	if err := c.EnableMCR(h, engineConfig); err != nil {
		return fmt.Errorf("package manager could not install docker-ee")
	}

	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, engineConfig commonconfig.MCRConfig) error {
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
