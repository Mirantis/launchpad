package enterpriselinux

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	log "github.com/sirupsen/logrus"
)

// Configurer is the EL family specific implementation of a host configurer.
type Configurer struct {
	configurer.LinuxConfigurer
}

// PrepareHost prepares the machine host by installing the needed base packages, and fixing any container issues.
func (c Configurer) PrepareHost(h configurer.Host) error {
	if err := c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip", "openssh"); err != nil {
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

	if isEC2 := c.isAWSInstance(h); !isEC2 {
		log.Debugf("%s: confirmed that this is not an AWS instance", h)
	} else if c.InstallPackage(h, "rh-amazon-rhui-client") == nil {
		log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", h)
	}

	// e.g. https://repos.mirantis.com/rhel/$releasever/$basearch/<update-channel>
	baseURL := fmt.Sprintf("%s/%s/%s/%s/%s", engineConfig.RepoURL, ver.ID, "$releasever", "$basearch", engineConfig.Channel)
	// e.g. https://repos.mirantis.com/oraclelinux/gpg
	gpgURL := fmt.Sprintf("%s/%s/gpg", engineConfig.RepoURL, ver.ID)
	elRepoFilePath := "/etc/yum.repos.d/docker-ee.repo"
	elRepoTemplate := `[mirantis]
name=Mirantis Container Runtime
baseurl=%s
enabled=1
gpgcheck=1
gpgkey=%s
`
	elRepo := fmt.Sprintf(elRepoTemplate, baseURL, gpgURL)

	if err := h.Sudo().FS().WriteFile(elRepoFilePath, []byte(elRepo), fs.FileMode(0o600)); err != nil {
		return fmt.Errorf("could not write Yum repo file for MCR")
	}

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
			return fmt.Errorf("remove docker-ee package: %w", err)
		}
	}

	return nil
}

// function to check if the host is an AWS instance - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c Configurer) isAWSInstance(h configurer.Host) bool {
	found, err := h.ExecOutput("curl -s -m 5 http://169.254.169.254/latest/dynamic/instance-identity/document | grep region")
	if err != nil {
		log.Debugf("%s: curl on local-linked AWS id document failed: %v", h, err)
		return false
	}
	return strings.Contains(found, "region")
}
