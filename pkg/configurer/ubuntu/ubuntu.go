package ubuntu

import (
	"fmt"
	"io/fs"
	"net/url"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/k0sproject/rig/v2/sh"
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

// mirantisKeyringPath is where the Mirantis APT signing key is installed.
const mirantisKeyringPath = "/usr/share/keyrings/mirantis-archive-keyring.gpg"

// gpgKeyImportScript downloads an armoured signing key and dearmors it into
// mirantisKeyringPath. The URL is referenced as the quoted positional "$1",
// never interpolated into this text, so bash expands it as data and never
// parses it as code. pipefail makes a failed download fail the command instead
// of silently installing an empty keyring.
const gpgKeyImportScript = `set -o pipefail; curl -fsSL -- "$1" | gpg --batch --yes --output ` + mirantisKeyringPath + ` --dearmor`

// gpgImportCommand builds the signing key import command for gpgURL.
//
// gpgURL derives from user-supplied config (spec.mcr.repoURL) and reaches a
// root-privileged shell. It is passed to bash as a positional argument rather
// than formatted into the script, so shell metacharacters in the URL stay
// literal data: a URL such as "https://host/$(id)/gpg" is fetched verbatim
// instead of starting a second command.
func gpgImportCommand(gpgURL string) string {
	return sh.Command("bash", "-c", gpgKeyImportScript, "launchpad-mcr-gpg", gpgURL)
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
	// engineConfig.RepoURL is user-supplied config. ParseRequestURI is a cheap
	// early reject for obvious junk, not the security control: "$", "(", ")"
	// and backticks are all legal URL characters and pass it. The control is
	// gpgImportCommand keeping the URL out of the script text.
	if _, err := url.ParseRequestURI(gpgURL); err != nil {
		return fmt.Errorf("invalid MCR repo URL %q: %w", gpgURL, err)
	}
	if err := h.Sudo().Exec(gpgImportCommand(gpgURL)); err != nil {
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
		return fmt.Errorf("package manager could not install containerd.io: %w", err)
	}
	installCmd, cmdErr := configurer.MCRInstallCommand(configurer.AptGet, engineConfig)
	if cmdErr != nil {
		return fmt.Errorf("could not build MCR install command: %w", cmdErr)
	}
	if err := h.Sudo().Exec(installCmd); err != nil {
		return fmt.Errorf("package manager could not install docker-ee: %w", err)
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
