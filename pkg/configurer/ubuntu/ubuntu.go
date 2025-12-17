package ubuntu

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	linux.Ubuntu
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "apt-utils", "socat", "iputils-ping"); err != nil {
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

	// WARNING: we do not check if this is ubuntu - make sure that it is if you use this code (we did it elsewhere)
	codename, _ := ver.ExtraFields["VERSION_CODENAME"] // e.g. jammy
	// e.g. https://repos.mirantis.com/rhel/$releasever/$basearch/<update-channel>
	baseUrl := fmt.Sprintf("%s/%s", engineConfig.RepoURL, ver.ID)
	gpgUrl := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)
	debRepoFilePath := "/etc/apt/sources.list.d/mirantis.sources"
	debRepoTemplate := `Types: deb
URIs: %s
Suites: %s
Architectures: amd64
Components: %s
Signed-by: /usr/share/keyrings/mirantis-archive-keyring.gpg
`
	debRepo := fmt.Sprintf(debRepoTemplate, baseUrl, codename, engineConfig.Channel)

	// https://docs.mirantis.com/mcr/25.0/install/mcr-linux/ubuntu.html instructions

	// 2. import the mirantis gpg key
	if err := h.Exec(fmt.Sprintf("sudo gpg --batch --yes --output /usr/share/keyrings/mirantis-archive-keyring.gpg --dearmor <<< $(curl -fsSL %s)", gpgUrl), exec.Sudo(h)); err != nil {
		return fmt.Errorf("Could not install the Mirantis Ubuntu GPG signing key")
	}

	// 4. write the repo file
	// @TODO check if we can use apt-add-repository instead of writing a file (probably has better validation)
	if err := c.WriteFile(h, debRepoFilePath, debRepo, "0600"); err != nil {
		return fmt.Errorf("Could not write APT repo file for MCR")
	}
	if err := h.Exec("apt update", exec.Sudo(h)); err != nil {
		return fmt.Errorf("could not update apt package info")
	}

	if err := c.InstallPackage(h, "docker-ee"); err != nil {
		return fmt.Errorf("Package manager could not install docker-ee")
	}
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

		if err := h.Exec("apt-get -y remove docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
			return fmt.Errorf("failed to uninstall docker-ee apt package: %w", err)
		}
	}

	return nil
}
