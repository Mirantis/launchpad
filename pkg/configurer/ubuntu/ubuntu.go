package ubuntu

import (
	"fmt"
	"io/fs"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	configurer.LinuxConfigurer
}

// PrepareHost prepares the machine host by installing the needed base packages, and fixing any container issues.
func (c Configurer) PrepareHost(h configurer.Host) error {
	if err := c.InstallPackage(h, "curl", "apt-utils", "socat", "iputils-ping"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}

	if c.IsContainer(h) {
		if err := c.FixContainer(h); err != nil {
			return fmt.Errorf("fix container: %w", err)
		}
	}
	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h configurer.Host, engineConfig commonconfig.MCRConfig) error {
	ver, verErr := configurer.ResolveLinux(h)
	if verErr != nil {
		return fmt.Errorf("could not discover Linux version information")
	}

	// WARNING: we do not confirm this is ubuntu - make sure that it is if you use this code (we did it elsewhere)
	codename := ver.ExtraFields["VERSION_CODENAME"] // e.g. jammy
	// e.g. https://repos.mirantis.com/rhel/$releasever/$basearch/<update-channel>
	baseURL := fmt.Sprintf("%s/%s", engineConfig.RepoURL, ver.ID)
	gpgURL := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)
	debRepoFilePath := "/etc/apt/sources.list.d/mirantis.sources"
	debRepoTemplate := `Types: deb
URIs: %s
Suites: %s
Architectures: amd64
Components: %s
Signed-by: /usr/share/keyrings/mirantis-archive-keyring.gpg
`
	debRepo := fmt.Sprintf(debRepoTemplate, baseURL, codename, engineConfig.Channel)

	// https://docs.mirantis.com/mcr/25.0/install/mcr-linux/ubuntu.html instructions

	// 2. import the mirantis gpg key
	if err := h.Sudo().Exec(fmt.Sprintf("gpg --batch --yes --output /usr/share/keyrings/mirantis-archive-keyring.gpg --dearmor <<< $(curl -fsSL %s)", gpgURL)); err != nil {
		return fmt.Errorf("could not install the Mirantis Ubuntu GPG signing key")
	}

	// 4. write the repo file
	// @TODO check if we can use apt-add-repository instead of writing a file (probably has better validation)
	if err := h.Sudo().FS().WriteFile(debRepoFilePath, []byte(debRepo), fs.FileMode(0o600)); err != nil {
		return fmt.Errorf("could not write APT repo file for MCR")
	}

	// InstallPackage refreshes the package indexes (apt-get update) before
	// installing, so the freshly written Mirantis repo is picked up here.
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
func (c Configurer) UninstallMCR(h configurer.Host, engineConfig commonconfig.MCRConfig) error {
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

		if err := c.RemovePackage(h, "docker-ee", "docker-ee-cli"); err != nil {
			return fmt.Errorf("failed to uninstall docker-ee package: %w", err)
		}
	}

	return nil
}
